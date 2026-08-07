extends Node
## Lobby singleton — WebSocket client to the central server's matchmaking channel.
##
## Maintains a long-lived connection to Settings.central_ws_url. The server pushes
## match-found events down this socket; the client sends queue/cancel commands up
## it. The connection is kept open across matches (only disconnect_lobby() tears it
## down) so the player can re-queue immediately after returning to the main menu.
##
## --- token-passing convention -------------------------------------------------
## WebSocketPeer in Godot 4 does not expose arbitrary request headers, so the JWT
## is passed as a query parameter: {central_ws_url}?token=<jwt>. The Go backend
## reads the token from the query string during the WS upgrade handshake (same
## code path it would use for the Authorization header) and validates it before
## completing the 101 Switching Protocols. This is an explicit client/server
## contract — see the central server's WS handler. If the token is missing or
## invalid the server rejects the upgrade and the peer ends up in STATE_CLOSED.

signal connected
signal disconnected(reason: String)
signal queued
signal match_found(match_id: String, gs_host: String, gs_port: int, players: Array, mode: String)
signal match_canceled
signal ws_error(message: String)

const RECONNECT_DELAY := 5.0   # seconds between retry attempts
const MAX_RETRIES := 3         # after this many retries, give up and emit disconnected

var _peer: WebSocketPeer = null
var _connected: bool = false
# True between connect_lobby() and disconnect_lobby(): drives auto-reconnect.
var _wants_connect: bool = false
var _retries: int = 0
var _retry_timer: float = 0.0

## The most recent match_found payload: {match_id, gs_host, gs_port, players, mode}.
## Populated on match_found and left in place across the match so the world /
## post-game flow can read the match_id (which the GS can't push via Game.config,
## since the host's _sync_config rpc overwrites the client's Game.config).
var last_match: Dictionary = {}


func _process(delta: float) -> void:
	if _peer == null:
		# Count down to the next reconnect attempt while we still want to be attached.
		if _wants_connect and not _connected and _retries < MAX_RETRIES and _retry_timer > 0.0:
			_retry_timer -= delta
			if _retry_timer <= 0.0:
				_open_socket()
		return
	_peer.poll()
	var state := _peer.get_ready_state()
	if state == WebSocketPeer.STATE_OPEN:
		if not _connected:
			_connected = true
			_retries = 0
			connected.emit()
		# Drain all inbound messages queued this frame.
		while _peer.get_available_packet_count() > 0:
			_handle_packet(_peer.get_packet())
	elif state == WebSocketPeer.STATE_CLOSED:
		var code := _peer.get_close_code()
		var reason := _peer.get_close_reason()
		var was_connected := _connected
		_connected = false
		_peer = null
		if was_connected:
			disconnected.emit("closed (code %d%s)" % [code, ": " + reason if reason != "" else ""])
		# Auto-reconnect unless the user explicitly hung up.
		if _wants_connect:
			if _retries < MAX_RETRIES:
				_retries += 1
				_retry_timer = RECONNECT_DELAY
			else:
				_wants_connect = false
				disconnected.emit("reconnect_failed")
	# STATE_CONNECTING / STATE_CLOSING: just wait for the next frame.

# ---------------------------------------------------------------- public API

func connect_lobby() -> void:
	if not Auth.is_logged_in():
		ws_error.emit("not_logged_in")
		return
	if _connected:
		return
	if _peer != null:
		# A connect attempt is already in flight — let it resolve.
		_wants_connect = true
		return
	_wants_connect = true
	_retries = 0
	_retry_timer = 0.0
	_open_socket()

func disconnect_lobby() -> void:
	_wants_connect = false
	_retries = 0
	_retry_timer = 0.0
	var was := _connected
	_connected = false
	if _peer != null:
		_peer.close()
		_peer = null
	if was:
		disconnected.emit("manual")

func is_connected() -> bool:
	return _connected

func queue_match(mode: String) -> void:
	if not _connected:
		ws_error.emit("not_connected")
		return
	_send({"type": "match_queue", "mode": mode})

func cancel_match() -> void:
	if not _connected:
		return
	_send({"type": "match_cancel"})

# ---------------------------------------------------------------- internals

func _open_socket() -> void:
	if not Auth.is_logged_in():
		ws_error.emit("not_logged_in")
		_wants_connect = false
		return
	var token := Auth.get_token()
	if token == "":
		ws_error.emit("no_token")
		_wants_connect = false
		return
	var base := Settings.central_ws_url
	# Append ?token= (or &token= if the URL already carries a query string).
	var sep := "&" if base.find("?") >= 0 else "?"
	var full_url := base + sep + "token=" + token.uri_encode()
	_peer = WebSocketPeer.new()
	var err := _peer.connect_to_url(full_url)
	if err != OK:
		ws_error.emit("connect_failed")
		_peer = null
		# Schedule a retry if we still want to be attached and haven't exhausted them.
		if _wants_connect and _retries < MAX_RETRIES:
			_retries += 1
			_retry_timer = RECONNECT_DELAY
		elif _wants_connect:
			_wants_connect = false
			disconnected.emit("reconnect_failed")
		return

func _send(msg: Dictionary) -> void:
	if _peer == null or _peer.get_ready_state() != WebSocketPeer.STATE_OPEN:
		return
	_peer.send_text(JSON.stringify(msg))

func _handle_packet(pkt: PackedByteArray) -> void:
	var text := pkt.get_string_from_utf8()
	if text == "":
		return
	var parsed = JSON.parse_string(text)
	if typeof(parsed) != TYPE_DICTIONARY:
		return
	var msg: Dictionary = parsed
	var type := String(msg.get("type", ""))
	match type:
		"queued":
			queued.emit()
		"match_found":
			var players = msg.get("players", [])
			if typeof(players) != TYPE_ARRAY:
				players = []
			var mid := String(msg.get("match_id", ""))
			var ghost := String(msg.get("gs_host", "127.0.0.1"))
			var gport := int(msg.get("gs_port", 27020))
			var mmode := String(msg.get("mode", ""))
			last_match = {
				"match_id": mid,
				"gs_host": ghost,
				"gs_port": gport,
				"players": players,
				"mode": mmode,
			}
			match_found.emit(mid, ghost, gport, players, mmode)
		"match_canceled":
			match_canceled.emit()
		"error":
			ws_error.emit(String(msg.get("message", "unknown")))
		_:
			# Unknown message type — ignore rather than spam the UI.
			pass
