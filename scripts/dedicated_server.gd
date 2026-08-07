extends Node
## Dedicated Game Server entry point.
##
## Runs only when BootChecker detected DEDICATED_SERVER mode (env OPENFIRE_GS_MATCH_ID set).
## Responsibilities:
##   1. Set Game.config for the requested mode (so the world loads the right map/rules).
##   2. Open the ENet server on gs_port (Net.host_game) — this process becomes peer 1.
##   3. Optionally fetch the expected player roster from the central server.
##   4. Start the match as soon as enough players are connected + ready.
##   5. Load world.tscn (MultiplayerSpawner needs it to replicate combatants).
##   6. On match end, POST the result to the central server and quit.
##
## This node owns NO scene tree visuals — it is a pure authority. The world scene
## (world.tscn) is loaded as a child via change_scene once the match starts.

const WORLD_SCENE := "res://scenes/world.tscn"
const RESULT_PATH_FMT := "%s/internal/matches/%s/result"

# Map central-server mode strings to the Game.Mode enum (see game.gd).
const _MODE_MAP := {
	"deathmatch": 0,      # Game.Mode.DEATHMATCH
	"coop": 1,            # Game.Mode.COOP
	"team_deathmatch": 2, # Game.Mode.TEAM_DEATHMATCH
	"domination": 3,      # Game.Mode.DOMINATION
	"battle_royale": 4,   # Game.Mode.BATTLE_ROYALE
	"adventure": 5,       # Game.Mode.ADVENTURE
}

var _match_id: String = ""
var _central_url: String = ""
var _internal_token: String = ""
var _http: HTTPRequest

# Match lifecycle state.
var _expected_user_ids: Array = []   # central roster (user_id strings), if fetched
var _match_started: bool = false
var _match_finished: bool = false
var _boot_time: float = 0.0

func _ready() -> void:
	_match_id = BootChecker.gs_match_id
	_central_url = BootChecker.central_url
	_internal_token = BootChecker.internal_token
	_boot_time = Time.get_ticks_msec()

	# 1) Configure Game.config for the requested mode.
	_apply_mode_config(BootChecker.gs_mode)
	Game.player_name = "DedicatedServer"

	# Set up the HTTPRequest early — needed by both the roster fetch and any
	# abandon-on-failure path below.
	_http = HTTPRequest.new()
	add_child(_http)

	# 2) Open the ENet server. host_game() makes us peer 1 (the authority).
	if not Net.host_game(BootChecker.gs_port):
		push_error("[GS] failed to bind port %d — exiting" % BootChecker.gs_port)
		_report_abandoned_and_quit()
		return
	print("[GS] listening on port %d, match=%s mode=%s" % [BootChecker.gs_port, _match_id, BootChecker.gs_mode])

	# 3) Optionally fetch the expected roster from the central server.
	_fetch_roster()

	# 4) Wire match lifecycle. Game.match_over fires on the authority when the match
	#    ends (frag limit / domination / last standing / mission complete-fail).
	Game.match_over.connect(_on_match_over)

	# 5) Heartbeat: poll for enough players, start when ready.
	set_process(true)

	# 6) Hard safety cap — if nobody connects within 5 minutes, abandon and quit.
	get_tree().create_timer(300.0).timeout.connect(_on_no_players_timeout)

## Translate the mode string into Game.config. We keep the existing defaults from
## game.gd but override mode + a sensible map. The lobby would normally pick these;
## for a dedicated server we use per-mode defaults.
func _apply_mode_config(mode_str: String) -> void:
	var mode_enum: int = _MODE_MAP.get(mode_str, 0)
	Game.config["mode"] = mode_enum
	# Pick a default map per mode (matches main_menu's MAPS / mission table).
	match mode_enum:
		Game.Mode.COOP:
			var missions := Missions.get_all()
			if not missions.is_empty():
				Game.config["mission_id"] = missions[0]["id"]
				Game.config["map"] = missions[0]["map"]
		Game.Mode.ADVENTURE:
			Game.config["map"] = "res://maps/terrain.tscn"
			Game.config["frag_limit"] = 0
			Game.config["bot_count"] = 12
		_:
			Game.config["map"] = "res://maps/arena.tscn"

## Ask the central server for the match's player roster. We use user_ids to know
## when everyone has joined. If this fails, we fall back to "start when at least
## one player is connected and the grace timer elapses" (handled in _process).
func _fetch_roster() -> void:
	if _central_url == "" or _match_id == "":
		return
	var url := "%s/internal/matches/%s" % [_central_url, _match_id]
	var err := _http.request(url, ["X-Internal-Token: " + _internal_token], HTTPClient.METHOD_GET, "")
	if err != OK:
		push_warning("[GS] roster fetch request failed (err %d) — falling back to grace-based start" % err)

func _http_request_completed(result: int, _response_code: int, _headers: PackedStringArray, body: PackedByteArray) -> void:
	if result != HTTPRequest.Result.RESULT_SUCCESS:
		return
	var text := body.get_string_from_utf8()
	var json := JSON.new()
	if json.parse(text) != OK:
		return
	var data: Dictionary = json.data
	if not data.has("players"):
		return
	_expected_user_ids.clear()
	for p in data["players"]:
		_expected_user_ids.append(String(p.get("user_id", "")))
	print("[GS] expected roster: %d players" % _expected_user_ids.size())

func _process(_delta: float) -> void:
	if _match_started or _match_finished:
		return
	# Start once enough players are connected. With a roster: all present. Without:
	# at least one player (grace handled by Net's existing 5s/70s timers in world.gd).
	var connected_peers: int = Net.players.size() - 1  # subtract self (peer 1)
	if _expected_user_ids.size() > 0:
		if connected_peers >= _expected_user_ids.size():
			_start_match()
	else:
		# Fallback: give a 15s grace for stragglers, then start if at least 1 joined.
		var elapsed := (Time.get_ticks_msec() - _boot_time) / 1000.0
		if connected_peers >= 1 and elapsed >= 15.0:
			_start_match()

func _start_match() -> void:
	if _match_started:
		return
	_match_started = true
	print("[GS] starting match (players=%d)" % (Net.players.size() - 1))
	# Net.start_match broadcasts config + _do_start.rpc to every peer. On the GS,
	# _do_start fires match_started locally — but the GS must ALSO load the world
	# scene so the MultiplayerSpawner can replicate combatants to clients (clients
	# load world.tscn on their own match_started, see main_menu.gd).
	Net.start_match()
	await get_tree().create_timer(0.2).timeout
	get_tree().change_scene_to_file(WORLD_SCENE)

## Called when Game.end_match fires (frag limit / domination / last standing /
## mission complete / mission failed / 30-min hard cap below). The GS is the
## authority, so this runs on the GS only.
func _on_match_over(result: Dictionary) -> void:
	if _match_finished:
		return
	_match_finished = true
	print("[GS] match over: %s" % result)
	# Build the result payload from the authoritative scoreboard.
	var players_payload: Array = []
	for id in Game.scores:
		var s: Dictionary = Game.scores[id]
		if s.get("is_bot", false):
			continue  # bots don't get persisted
		var uname: String = Net.get_player_name(id)
		players_payload.append({
			"user_id": uname,  # TODO: map peer id -> real user_id once GS receives it
			"username": uname,
			"kills": int(s.get("kills", 0)),
			"deaths": int(s.get("deaths", 0)),
			"placement": 0,
			"team": int(s.get("team", 0)),
		})
	_post_result(players_payload)
	# Let clients see the result screen, then shut down.
	await get_tree().create_timer(5.0).timeout
	get_tree().quit()

func _post_result(players_payload: Array) -> void:
	if _central_url == "" or _match_id == "":
		return
	var url := RESULT_PATH_FMT % [_central_url, _match_id]
	var body := JSON.stringify({
		"status": "finished",
		"players": players_payload,
	})
	var hdrs := PackedStringArray([
		"X-Internal-Token: " + _internal_token,
		"Content-Type: application/json",
	])
	var err := _http.request(url, hdrs, HTTPClient.METHOD_POST, body)
	if err != OK:
		push_error("[GS] failed to POST result (err %d)" % err)

func _report_abandoned_and_quit() -> void:
	if _central_url == "" or _match_id == "":
		get_tree().quit()
		return
	var url := RESULT_PATH_FMT % [_central_url, _match_id]
	var body := JSON.stringify({"status": "abandoned", "players": []})
	var hdrs := PackedStringArray([
		"X-Internal-Token: " + _internal_token,
		"Content-Type: application/json",
	])
	_http.request(url, hdrs, HTTPClient.METHOD_POST, body)
	await get_tree().create_timer(1.0).timeout
	get_tree().quit()

func _on_no_players_timeout() -> void:
	if not _match_started and not _match_finished:
		print("[GS] no players after 5 min — abandoning")
		_report_abandoned_and_quit()

func _notification(what: int) -> void:
	# Process killed (SIGTERM from central server) — mark abandoned if not finished.
	if what == NOTIFICATION_WM_CLOSE_REQUEST and not _match_finished:
		_report_abandoned_and_quit()
