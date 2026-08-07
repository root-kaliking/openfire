extends Node
## Auth singleton — account / JWT client for the central server.
##
## Persists the login state (token + user_id + username) to user://auth.cfg so a
## returning player doesn't have to log in every launch. The token is sent as a
## Bearer header on REST calls and as a ?token= query param on the lobby WebSocket
## (see lobby.gd for the WS token-passing convention).

signal logged_in(user_id: String, username: String)
signal logged_out
signal login_failed(reason: String)

const AUTH_PATH := "user://auth.cfg"
const SECTION := "auth"

var _token: String = ""
var _user_id: String = ""
var _username: String = ""

var _http: HTTPRequest = null
# The HTTPRequest node holds a single in-flight request; guard against overlapping
# login/register calls so the second one doesn't clobber the first's await.
var _busy: bool = false


func _ready() -> void:
	_http = HTTPRequest.new()
	add_child(_http)
	try_restore_session()

# ---------------------------------------------------------------- session persistence

func _save_session() -> void:
	var cfg := ConfigFile.new()
	cfg.set_value(SECTION, "token", _token)
	cfg.set_value(SECTION, "user_id", _user_id)
	cfg.set_value(SECTION, "username", _username)
	cfg.save(AUTH_PATH)

func _clear_local_state() -> void:
	_token = ""
	_user_id = ""
	_username = ""
	# Drop the persisted file so a stale token isn't restored next launch.
	var da := DirAccess.open("user://")
	if da != null and da.file_exists("auth.cfg"):
		da.remove("auth.cfg")

## On startup, read the saved token (if any). The central server has no /api/me
## endpoint in the current contract, so we don't validate remotely — we just restore
## the local state and let the lobby / match calls fail later if the token has
## actually expired. Returns true if a token was restored.
func try_restore_session() -> bool:
	var cfg := ConfigFile.new()
	if cfg.load(AUTH_PATH) != OK:
		return false
	_token = String(cfg.get_value(SECTION, "token", ""))
	_user_id = String(cfg.get_value(SECTION, "user_id", ""))
	_username = String(cfg.get_value(SECTION, "username", ""))
	if _token == "":
		_token = ""
		_user_id = ""
		_username = ""
		return false
	return true

# ---------------------------------------------------------------- public API

func is_logged_in() -> bool:
	return _token != ""

func get_token() -> String:
	return _token

func get_user_id() -> String:
	return _user_id

func get_username() -> String:
	return _username

func register(username: String, password: String) -> void:
	_do_auth_request("/api/auth/register", username, password)

func login(username: String, password: String) -> void:
	_do_auth_request("/api/auth/login", username, password)

func logout() -> void:
	if not is_logged_in():
		return
	_clear_local_state()
	logged_out.emit()

## Returns the full Authorization header value for Bearer-token REST requests, or
## "" when not logged in (callers should check is_logged_in() first).
func auth_header() -> String:
	if _token == "":
		return ""
	return "Authorization: Bearer " + _token

# ---------------------------------------------------------------- HTTP

func _do_auth_request(path: String, username: String, password: String) -> void:
	if _busy:
		login_failed.emit("busy")
		return
	username = username.strip_edges()
	if username == "" or password == "":
		login_failed.emit("empty")
		return
	var body := JSON.stringify({"username": username, "password": password})
	var headers := PackedStringArray(["Content-Type: application/json"])
	var url := Settings.central_url + path
	var err := _http.request(url, headers, HTTPClient.METHOD_POST, body)
	if err != OK:
		login_failed.emit("network")
		return
	_busy = true
	var result: Array = await _http.request_completed
	_busy = false
	_handle_auth_response(result)

func _handle_auth_response(result: Array) -> void:
	# result = [result_code, response_code, headers, body]
	var rcode: int = result[0]
	var http_code: int = result[1]
	var body_bytes: PackedByteArray = result[3]
	if rcode != HTTPRequest.RESULT_SUCCESS:
		login_failed.emit("network")
		return
	var body_text := body_bytes.get_string_from_utf8()
	var parsed = JSON.parse_string(body_text)
	if typeof(parsed) != TYPE_DICTIONARY:
		login_failed.emit("network")
		return
	if http_code < 200 or http_code >= 300:
		# Try to surface the backend's own error string (e.g. "taken", "invalid_credentials").
		var reason := String(parsed.get("error", ""))
		if reason == "":
			reason = "credentials"
		login_failed.emit(reason)
		return
	var user: Dictionary = parsed.get("user", {})
	_token = String(parsed.get("token", ""))
	_user_id = String(user.get("id", ""))
	_username = String(user.get("username", ""))
	if _token == "":
		login_failed.emit("network")
		return
	_save_session()
	logged_in.emit(_user_id, _username)
