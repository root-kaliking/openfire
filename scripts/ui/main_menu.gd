extends Control
## Main menu + lobby. Configure a match, then Host / Join (by IP) / Solo-vs-bots.
## The host captures settings into Game.config; Net replicates them to clients.

const MAPS := [
	{ "name": "Arena", "path": "res://maps/arena.tscn" },
	{ "name": "Facility", "path": "res://maps/facility.tscn" },
	{ "name": "Highlands", "path": "res://maps/highlands.tscn" },
	{ "name": "Warehouse", "path": "res://maps/warehouse.tscn" },
	{ "name": "Ruins", "path": "res://maps/ruins.tscn" },
	{ "name": "Compound", "path": "res://maps/compound.tscn" },
	{ "name": "Outpost (huge, vehicles)", "path": "res://maps/outpost.tscn" },
	{ "name": "Badlands (huge, vehicles)", "path": "res://maps/badlands.tscn" },
	{ "name": "Wasteland (massive, battle royale)", "path": "res://maps/wasteland.tscn" },
]
const SKILLS := [
	{ "name": "Easy", "value": 0.6 },
	{ "name": "Normal", "value": 1.0 },
	{ "name": "Hard", "value": 1.4 },
]
# Adventure uses one procedurally-generated terrain map, sized by map_size + seed.
const ADVENTURE_MAP := "res://maps/terrain.tscn"

# Saved map templates: each is a fixed (seed + size + theme + climate) that regenerates
# the exact same world every time. "Custom" uses the fields below instead. size:
# 0 Tiny, 1 Small, 2 Medium, 3 Large.
const MAP_PRESETS := [
	{"name": "Custom (use fields below)", "preset": false},
	{"name": "★ Showoff — Grand Vista (huge)", "preset": true, "seed": 820447, "size": 3,
		"theme": "a majestic valley of snow-capped peaks, winding rivers, lakes and deep forests", "climate": "verdant"},
	{"name": "Test — Frozen North", "preset": true, "seed": 30211, "size": 2, "theme": "frozen arctic tundra", "climate": "frozen"},
	{"name": "Test — Desert Wastes", "preset": true, "seed": 30222, "size": 2, "theme": "scorching desert wasteland", "climate": "desert"},
	{"name": "Test — Tropic Isles", "preset": true, "seed": 30233, "size": 2, "theme": "tropical island archipelago", "climate": "isles"},
	{"name": "Test — Volcanic Ashlands", "preset": true, "seed": 30244, "size": 2, "theme": "volcanic ashlands", "climate": "volcanic"},
	{"name": "Test — Verdant Lowlands", "preset": true, "seed": 30255, "size": 2, "theme": "lush green wilderness", "climate": "verdant"},
]

@onready var setup_panel: Control = %SetupPanel
@onready var lobby_panel: Control = %LobbyPanel
@onready var name_edit: LineEdit = %NameEdit
@onready var mode_option: OptionButton = %ModeOption
@onready var map_row: Control = %MapRow
@onready var map_option: OptionButton = %MapOption
@onready var mission_row: Control = %MissionRow
@onready var mission_option: OptionButton = %MissionOption
@onready var frag_row: Control = %FragRow
@onready var frag_spin: SpinBox = %FragSpin
@onready var bots_spin: SpinBox = %BotsSpin
@onready var skill_option: OptionButton = %SkillOption
@onready var ip_edit: LineEdit = %IpEdit
@onready var status_label: Label = %StatusLabel
@onready var lobby_players: VBoxContainer = %LobbyPlayers
@onready var lobby_summary: Label = %LobbySummary
@onready var start_button: Button = %StartButton
@onready var options_panel: Control = %OptionsPanel
@onready var mission_points_row: Control = %MissionPointsRow
@onready var mission_points_spin: SpinBox = %MissionPointsSpin
@onready var seed_row: Control = %SeedRow
@onready var seed_edit: LineEdit = %SeedEdit
@onready var theme_row: Control = %ThemeRow
@onready var theme_edit: LineEdit = %ThemeEdit
@onready var map_size_row: Control = %MapSizeRow
@onready var map_size_option: OptionButton = %MapSizeOption
@onready var template_row: Control = %TemplateRow
@onready var template_option: OptionButton = %TemplateOption
@onready var inv_key_option: OptionButton = %InvKeyOption
@onready var character_row: Control = %CharacterRow
@onready var char_label: Label = %CharLabel
@onready var character_panel: Control = %CharacterPanel
@onready var create_panel: Control = %CreatePanel
@onready var char_list: VBoxContainer = %CharList

# --- online (central server) UI ----------------------------------------------
# Online-panel widgets are built in code so the .tscn stays untouched.
var _online_match: bool = false        # true between match_found and scene switch / failure
var _queuing: bool = false             # true while waiting for match_found (toggles PLAY ONLINE)
var _pending_queue: bool = false       # user clicked PLAY ONLINE before the lobby connected
var _history_http: HTTPRequest = null
var _setup_vbox: VBoxContainer = null
var _auth_row: Control = null
var _username_edit: LineEdit = null
var _password_edit: LineEdit = null
var _login_btn: Button = null
var _register_btn: Button = null
var _auth_status: Label = null
var _logged_in_row: Control = null
var _logged_in_label: Label = null
var _logout_btn: Button = null
var _play_online_btn: Button = null
var _history_btn: Button = null
var _match_status: Label = null
var _history_panel: Control = null
var _history_status: Label = null
var _history_list: VBoxContainer = null

# Selectable inventory keys for Adventure (label + keycode).
const INV_KEYS := [
	{ "name": "Tab", "code": KEY_TAB },
	{ "name": "I", "code": KEY_I },
	{ "name": "B", "code": KEY_B },
	{ "name": "C", "code": KEY_C },
]

# Embedded llama.cpp models (Qwen2.5 Instruct, Q4_K_M GGUF). Downloaded on first
# Adventure start into user://models/. Bigger = better stories, larger download.
const AI_MODELS := [
	{
		"name": "Tiny — Qwen2.5 0.5B (~0.4 GB)",
		"file": "Qwen2.5-0.5B-Instruct-Q4_K_M.gguf",
		"url": "https://huggingface.co/bartowski/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/Qwen2.5-0.5B-Instruct-Q4_K_M.gguf",
	},
	{
		"name": "Small — Qwen2.5 1.5B (~1 GB)",
		"file": "Qwen2.5-1.5B-Instruct-Q4_K_M.gguf",
		"url": "https://huggingface.co/bartowski/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/Qwen2.5-1.5B-Instruct-Q4_K_M.gguf",
	},
	{
		"name": "Medium — Qwen2.5 3B (~2 GB)",
		"file": "Qwen2.5-3B-Instruct-Q4_K_M.gguf",
		"url": "https://huggingface.co/bartowski/Qwen2.5-3B-Instruct-GGUF/resolve/main/Qwen2.5-3B-Instruct-Q4_K_M.gguf",
	},
	{
		"name": "Huge — Qwen2.5 7B (~4.7 GB)",
		"file": "Qwen2.5-7B-Instruct-Q4_K_M.gguf",
		"url": "https://huggingface.co/bartowski/Qwen2.5-7B-Instruct-GGUF/resolve/main/Qwen2.5-7B-Instruct-Q4_K_M.gguf",
	},
]

func _ready() -> void:
	Input.mouse_mode = Input.MOUSE_MODE_VISIBLE
	Net.players_changed.connect(_refresh_lobby)
	Net.connection_succeeded.connect(_on_connected)
	Net.connection_failed.connect(_on_failed)
	Net.server_disconnected.connect(_on_server_disconnected)
	Net.match_started.connect(_on_match_started)

	# --- online (central server) wiring ---
	Auth.logged_in.connect(_on_auth_logged_in)
	Auth.logged_out.connect(_on_auth_logged_out)
	Auth.login_failed.connect(_on_auth_login_failed)
	Lobby.connected.connect(_on_lobby_connected)
	Lobby.disconnected.connect(_on_lobby_disconnected)
	Lobby.queued.connect(_on_lobby_queued)
	Lobby.match_found.connect(_on_lobby_match_found)
	Lobby.match_canceled.connect(_on_lobby_match_canceled)
	Lobby.ws_error.connect(_on_lobby_ws_error)
	# BootChecker redirects dedicated servers before the menu loads; connect defensively
	# (the stub doesn't define the signal yet) and guard the menu if we're not a client.
	if BootChecker.has_signal("boot_done"):
		BootChecker.boot_done.connect(_on_boot_done)
	_history_http = HTTPRequest.new()
	add_child(_history_http)
	_build_online_panel()
	_build_history_panel()
	_refresh_auth_ui()
	# Auto-attach the lobby WebSocket if we have a saved session, so the player can
	# queue the moment they click PLAY ONLINE.
	if Auth.is_logged_in() and not Lobby.is_lobby_connected():
		Lobby.connect_lobby()

	name_edit.text = Game.player_name
	%VersionLabel.text = "v" + str(ProjectSettings.get_setting("application/config/version", "0.0.0"))
	mode_option.clear()
	mode_option.add_item("Deathmatch")
	mode_option.add_item("Co-op")
	mode_option.add_item("Team Deathmatch")
	mode_option.add_item("Domination")
	mode_option.add_item("Battle Royale")
	mode_option.add_item("Adventure")
	mode_option.selected = Game.Mode.ADVENTURE  # Adventure is the default mode
	template_option.clear()
	for p in MAP_PRESETS:
		template_option.add_item(String(p["name"]))
	template_option.selected = 0
	template_option.item_selected.connect(func(_i): _on_mode_changed(mode_option.selected))
	map_size_option.clear()
	for size_name in ["Tiny", "Small", "Medium", "Large"]:
		map_size_option.add_item(size_name)
	map_size_option.selected = 2   # Medium
	map_option.clear()
	for m in MAPS:
		map_option.add_item(m["name"])
	skill_option.clear()
	for s in SKILLS:
		skill_option.add_item(s["name"])
	skill_option.selected = 1
	mission_option.clear()
	for mission in Missions.get_all():
		mission_option.add_item(mission["name"])

	%HostButton.pressed.connect(_on_host)
	%JoinButton.pressed.connect(_on_join)
	%SoloButton.pressed.connect(_on_solo)
	%QuitButton.pressed.connect(func(): get_tree().quit())
	mode_option.item_selected.connect(_on_mode_changed)
	start_button.pressed.connect(func(): Net.start_match())
	%BackButton.pressed.connect(_on_back)

	# Character screens (Adventure).
	%CreateKit.clear()
	for kit_id in Characters.KIT_IDS:
		%CreateKit.add_item(Characters.kit_name(kit_id))
	%CharBtn.pressed.connect(_show_characters)
	%ContinueBtn.pressed.connect(_on_continue)
	%CharBackBtn.pressed.connect(_show_setup)
	%NewCharBtn.pressed.connect(_show_create)
	%DeleteCharBtn.pressed.connect(_on_delete_character)
	%CreateConfirm.pressed.connect(_on_create_confirm)
	%CreateCancel.pressed.connect(_show_characters)
	_update_char_label()

	# Click sound on every button.
	for b in find_children("*", "Button", true):
		b.pressed.connect(func(): Audio.play_ui("res://assets/audio/ui_click.ogg", -4.0))

	_setup_options()
	_on_mode_changed(mode_option.selected)
	_show_setup()

	# Gate play on the AI download: the game generates world assets via ComfyUI, so don't let
	# anyone start until the bundle + models are downloaded and ComfyUI is answering.
	if not ComfyUI.server_checked.is_connected(_on_ai_ready_changed):
		ComfyUI.server_checked.connect(_on_ai_ready_changed)
		ComfyUI.setup_status.connect(_on_ai_ready_changed_s)
	_update_play_gate()

func _setup_options() -> void:
	%SensSlider.value = Settings.mouse_sensitivity
	%VolSlider.value = Settings.master_volume
	%FovSlider.value = Settings.fov
	inv_key_option.clear()
	var sel := 0
	for i in INV_KEYS.size():
		inv_key_option.add_item(INV_KEYS[i]["name"])
		if int(INV_KEYS[i]["code"]) == Settings.inventory_keycode:
			sel = i
	inv_key_option.selected = sel
	# Graphics quality preset.
	%QualityOption.clear()
	for qn in Settings.QUALITY_NAMES:
		%QualityOption.add_item(qn)
	%QualityOption.selected = clampi(Settings.quality, 0, 2)
	%QualityOption.item_selected.connect(_on_quality_changed)
	# Embedded AI model preset: match the saved file to a preset (default Small).
	%LlmEmbedOption.clear()
	var msel := 1
	for i in AI_MODELS.size():
		%LlmEmbedOption.add_item(String(AI_MODELS[i]["name"]))
		if String(AI_MODELS[i]["file"]) == Settings.llm_model_file:
			msel = i
	%LlmEmbedOption.selected = msel
	%LlmEmbedOption.item_selected.connect(_on_llm_embed_changed)
	_update_option_labels()
	%SensSlider.value_changed.connect(_on_sens_changed)
	%VolSlider.value_changed.connect(_on_vol_changed)
	%FovSlider.value_changed.connect(_on_fov_changed)
	inv_key_option.item_selected.connect(_on_inv_key_changed)
	%LlmEndpointEdit.text = Settings.llm_endpoint
	%LlmModelEdit.text = Settings.llm_model
	%LlmEndpointEdit.text_changed.connect(_on_llm_endpoint_changed)
	%LlmModelEdit.text_changed.connect(_on_llm_model_changed)
	# Debug mode toggle: enables the in-game [0] cheat menu (solo games only).
	var dbg := CheckButton.new()
	dbg.text = "Debug mode  ·  press [0] in a solo game"
	dbg.button_pressed = Settings.debug_mode
	dbg.toggled.connect(_on_debug_toggled)
	var vbox: Node = %OptionsBackButton.get_parent()
	vbox.add_child(dbg)
	vbox.move_child(dbg, %OptionsBackButton.get_index())   # sit just above the Back button
	_build_comfyui_options(vbox)
	%OptionsButton.pressed.connect(_show_options)
	%OptionsBackButton.pressed.connect(_show_setup)

func _on_debug_toggled(on: bool) -> void:
	Settings.debug_mode = on
	Settings.save()

var _comfy_status: Label = null
var _comfy_dl_bar: ProgressBar = null

## Opt-in ComfyUI asset bridge controls: enable, server endpoint, checkpoint, and a button
## to pre-bake a sample themed prop library into user://generated/ (needs a running ComfyUI).
func _build_comfyui_options(vbox: Node) -> void:
	var head := Label.new()
	head.text = "AI assets (ComfyUI)"
	vbox.add_child(head)
	var ep := LineEdit.new()
	ep.placeholder_text = "ComfyUI endpoint (http://127.0.0.1:8188)"
	ep.text = Settings.comfyui_endpoint
	ep.text_changed.connect(func(t): Settings.comfyui_endpoint = t.strip_edges(); Settings.save())
	vbox.add_child(ep)
	# The model auto-downloads into the bundled checkpoints folder (next to the game).
	var dl := Button.new()
	dl.text = "Download the AI model (~4 GB, first time)"
	dl.pressed.connect(_on_download_model)
	vbox.add_child(dl)
	_comfy_dl_bar = ProgressBar.new()
	_comfy_dl_bar.min_value = 0.0
	_comfy_dl_bar.max_value = 100.0
	_comfy_dl_bar.custom_minimum_size = Vector2(420, 18)
	_comfy_dl_bar.visible = false
	vbox.add_child(_comfy_dl_bar)
	_comfy_status = Label.new()
	_comfy_status.text = ""
	_comfy_status.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	_comfy_status.custom_minimum_size.x = 420
	vbox.add_child(_comfy_status)
	for n in [head, ep, dl, _comfy_dl_bar, _comfy_status]:
		vbox.move_child(n, %OptionsBackButton.get_index())
	# Reflect the automatic first-run setup (bundle download, model downloads, "starting…")
	# without the user clicking anything.
	if not ComfyUI.setup_status.is_connected(_on_comfy_setup_status):
		ComfyUI.setup_status.connect(_on_comfy_setup_status)
	if ComfyUI.setup_message != "":
		_on_comfy_setup_status(ComfyUI.setup_message, ComfyUI.setup_fraction)

var _ai_gate_label: Label = null

## Disable Host/Join/Solo until the AI stack is downloaded and ComfyUI is answering, with a
## status line above the buttons explaining the wait.
func _update_play_gate() -> void:
	var ready := ComfyUI.is_ready()
	for b in [%HostButton, %JoinButton, %SoloButton]:
		if b != null:
			b.disabled = not ready
	if _ai_gate_label == null:
		_ai_gate_label = Label.new()
		_ai_gate_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
		_ai_gate_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
		_ai_gate_label.custom_minimum_size = Vector2(360, 0)   # else it collapses to a 1-char column
		_ai_gate_label.size_flags_horizontal = Control.SIZE_SHRINK_CENTER
		_ai_gate_label.modulate = Color(1, 0.85, 0.4)
		# Add to the VBox ABOVE the button row (HostButton's parent is the Buttons HBox — adding
		# there would put the label on the same line as the buttons).
		var row := %HostButton.get_parent()
		var container := row.get_parent()
		container.add_child(_ai_gate_label)
		container.move_child(_ai_gate_label, row.get_index())
	_ai_gate_label.visible = not ready
	if not ready:
		var msg := ComfyUI.setup_message if ComfyUI.setup_message != "" else "Preparing AI (downloading models)…"
		_ai_gate_label.text = "⏳ %s" % msg

func _on_ai_ready_changed(_ok: bool) -> void:
	_update_play_gate()

func _on_ai_ready_changed_s(_message: String, _fraction: float) -> void:
	_update_play_gate()

func _on_comfy_setup_status(message: String, fraction: float) -> void:
	if _comfy_status == null:
		return
	if message == "":
		_comfy_status.text = "AI ready."
		if _comfy_dl_bar != null:
			_comfy_dl_bar.visible = false
		return
	_comfy_status.text = message
	if _comfy_dl_bar != null:
		if fraction >= 0.0:
			_comfy_dl_bar.visible = true
			_comfy_dl_bar.value = fraction * 100.0
		else:
			_comfy_dl_bar.visible = false

func _on_download_model() -> void:
	if not ComfyUI.model_ready.is_connected(_on_model_ready_status):
		ComfyUI.model_ready.connect(_on_model_ready_status)
		ComfyUI.model_progress.connect(_on_model_progress)
	if ComfyUI.has_local_model():
		_comfy_status.text = "Model already present — you're set."
		return
	_comfy_status.text = "Downloading model… (a few GB, one time)"
	if _comfy_dl_bar != null:
		_comfy_dl_bar.visible = true
		_comfy_dl_bar.value = 0.0
	ComfyUI.download_model()

func _on_model_progress(_frac: float, downloaded: int, total: int) -> void:
	var mb := func(n: int) -> String: return "%.0f MB" % (float(n) / 1048576.0)
	# HuggingFace's LFS redirects make the HTTP Content-Length (`total`) unreliable — often a
	# fraction of the real size. Only trust it when it's clearly the whole file; otherwise use
	# the known model size. `downloaded` (actual bytes written) is always correct.
	var real_total := total if (total > downloaded and total > 1073741824) else int(Settings.comfyui_model_size)
	if real_total > 0:
		var pct := clampf(float(downloaded) / float(real_total) * 100.0, 0.0, 99.0)
		if _comfy_dl_bar != null:
			_comfy_dl_bar.value = pct
		_comfy_status.text = "Downloading model…  %d%%   (%s / %s)" % [int(pct), mb.call(downloaded), mb.call(real_total)]
	else:
		if _comfy_dl_bar != null:
			_comfy_dl_bar.value = fmod(float(downloaded) / 1048576.0, 100.0)
		_comfy_status.text = "Downloading model…  %s" % mb.call(downloaded)

func _on_model_ready_status(_ok: bool, message: String) -> void:
	if _comfy_dl_bar != null:
		_comfy_dl_bar.visible = false
	_comfy_status.text = message

func _on_llm_embed_changed(idx: int) -> void:
	var m: Dictionary = AI_MODELS[clampi(idx, 0, AI_MODELS.size() - 1)]
	Settings.llm_model_url = String(m["url"])
	Settings.llm_model_file = String(m["file"])
	Settings.save()

func _on_llm_endpoint_changed(t: String) -> void:
	Settings.llm_endpoint = t.strip_edges()
	Settings.save()

func _on_llm_model_changed(t: String) -> void:
	Settings.llm_model = t.strip_edges()
	Settings.save()

func _on_inv_key_changed(idx: int) -> void:
	Settings.inventory_keycode = int(INV_KEYS[clampi(idx, 0, INV_KEYS.size() - 1)]["code"])
	Settings.save()

func _on_sens_changed(v: float) -> void:
	Settings.mouse_sensitivity = v
	_apply_and_save()

func _on_vol_changed(v: float) -> void:
	Settings.master_volume = v
	_apply_and_save()

func _on_fov_changed(v: float) -> void:
	Settings.fov = v
	_apply_and_save()

func _on_quality_changed(idx: int) -> void:
	Settings.quality = clampi(idx, 0, 2)
	_apply_and_save()   # takes effect on the next match (environment built on load)

func _apply_and_save() -> void:
	_update_option_labels()
	Settings.apply()
	Settings.save()

func _update_option_labels() -> void:
	%SensValue.text = "%.2f" % Settings.mouse_sensitivity
	%VolValue.text = "%d%%" % int(round(Settings.master_volume * 100))
	%FovValue.text = "%d" % int(Settings.fov)

func _show_options() -> void:
	setup_panel.visible = false
	lobby_panel.visible = false
	options_panel.visible = true
	character_panel.visible = false
	create_panel.visible = false

func _on_mode_changed(_idx: int) -> void:
	var coop := mode_option.selected == 1
	var br := mode_option.selected == 4  # Battle Royale: no frag limit, last one alive wins
	var adventure := mode_option.selected == 5
	map_row.visible = not coop and not adventure      # Adventure picks a size, not a map
	frag_row.visible = not coop and not br and not adventure
	mission_row.visible = coop
	mission_points_row.visible = adventure
	# A chosen template pins seed/size/theme, so hide those fields then.
	var custom: bool = adventure and template_option.selected == 0
	template_row.visible = adventure
	seed_row.visible = custom
	map_size_row.visible = custom
	theme_row.visible = custom
	character_row.visible = adventure
	# Adventure hides the manual NPC count, but keeps the difficulty (Bot skill) selector.
	bots_spin.get_parent().visible = not adventure
	if coop and Missions.get_all().is_empty():
		status_label.text = "No missions found in res://missions/"

func _capture_config() -> void:
	Game.player_name = name_edit.text.strip_edges()
	if Game.player_name == "":
		Game.player_name = "Player"
	var coop := mode_option.selected == 1
	var adventure := mode_option.selected == 5
	# Option order matches the Mode enum (0=Deathmatch … 5=Adventure).
	Game.config["mode"] = mode_option.selected
	Game.config["bot_count"] = int(bots_spin.value)
	Game.config["bot_skill"] = SKILLS[skill_option.selected]["value"]
	if coop:
		var missions := Missions.get_all()
		if not missions.is_empty():
			var m: Dictionary = missions[clampi(mission_option.selected, 0, missions.size() - 1)]
			Game.config["mission_id"] = m["id"]
			Game.config["map"] = m["map"]
	elif adventure:
		Game.config.erase("climate")        # fresh world -> re-resolve from the theme
		Game.continue_data = {}
		Game.config["mission_points"] = int(mission_points_spin.value)
		Game.config["map"] = ADVENTURE_MAP   # terrain.gd reads map_size + seed
		Game.config["frag_limit"] = 0
		var preset: Dictionary = MAP_PRESETS[clampi(template_option.selected, 0, MAP_PRESETS.size() - 1)]
		if bool(preset.get("preset", false)):
			# A saved template: load its exact world (seed + size + theme + pinned climate).
			Game.config["seed"] = int(preset["seed"])
			Game.config["map_size"] = int(preset["size"])
			Game.config["theme"] = String(preset["theme"])
			Game.config["climate"] = String(preset["climate"])
		else:
			Game.config["map_size"] = map_size_option.selected
			Game.config["seed"] = _parse_seed(seed_edit.text.strip_edges())
			Game.config["theme"] = theme_edit.text.strip_edges()
		# NPC count is fixed (scaled by the world); difficulty comes from the skill
		# selector (bot_skill is already set from skill_option above).
		Game.config["bot_count"] = 12
	else:
		Game.config["map"] = MAPS[map_option.selected]["path"]
		Game.config["frag_limit"] = int(frag_spin.value)

## Resolve the seed field: blank = random, a plain integer = itself, else hashed.
func _parse_seed(txt: String) -> int:
	if txt == "":
		return randi()
	if txt.is_valid_int():
		return int(txt)
	return hash(txt)

# ---------------------------------------------------------------- buttons

func _on_host() -> void:
	_capture_config()
	if Net.host_game():
		status_label.text = "Hosting on port %d. Share your LAN IP with friends." % Net.DEFAULT_PORT
		_show_lobby()
	else:
		status_label.text = "Failed to host (port in use?)."

func _on_solo() -> void:
	_capture_config()
	if not Game.is_adventure():
		Game.config["bot_count"] = maxi(1, int(bots_spin.value))
	if Net.host_game():
		Net.start_match()
	else:
		status_label.text = "Failed to start solo match."

func _on_join() -> void:
	Game.player_name = name_edit.text.strip_edges()
	if Game.player_name == "":
		Game.player_name = "Player"
	var ip := ip_edit.text.strip_edges()
	if ip == "":
		ip = "127.0.0.1"
	if Net.join_game(ip):
		status_label.text = "Connecting to %s…" % ip
	else:
		status_label.text = "Could not start client."

func _on_back() -> void:
	Net.disconnect_net()
	_show_setup()

# ---------------------------------------------------------------- net callbacks

func _on_connected() -> void:
	if _online_match:
		# Connected to a dedicated game server via matchmaking — don't show the custom
		# lobby panel; just wait for the GS to start the match.
		status_label.text = "Connected to game server, waiting for match to start…"
		_match_status.text = "已连接游戏服务器,等待对局开始..."
		return
	status_label.text = "Connected. Waiting for host to start…"
	_show_lobby()

func _on_failed() -> void:
	if _online_match:
		_online_match = false
		_play_online_btn.disabled = not Auth.is_logged_in()
		_match_status.text = "连接游戏服务器失败"
		_match_status.modulate = Color(1, 0.6, 0.6)
	status_label.text = "Connection failed."
	_show_setup()

func _on_server_disconnected() -> void:
	if _online_match:
		_online_match = false
		_play_online_btn.disabled = not Auth.is_logged_in()
		_match_status.text = "与游戏服务器断开"
		_match_status.modulate = Color(1, 0.6, 0.6)
	status_label.text = "Disconnected from host."
	_show_setup()

func _on_match_started() -> void:
	get_tree().change_scene_to_file("res://scenes/world.tscn")

# ---------------------------------------------------------------- screens

func _show_setup() -> void:
	setup_panel.visible = true
	lobby_panel.visible = false
	options_panel.visible = false
	character_panel.visible = false
	create_panel.visible = false
	_update_char_label()

func _show_lobby() -> void:
	setup_panel.visible = false
	options_panel.visible = false
	character_panel.visible = false
	create_panel.visible = false
	lobby_panel.visible = true
	start_button.visible = Net.is_host()
	_refresh_lobby()

# ---------------------------------------------------------------- characters

func _update_char_label() -> void:
	char_label.text = String(Characters.current.get("name", "(none)")) if Characters.has_current() else "(none)"
	# Continue is only offered when the chosen character has a saved adventure.
	%ContinueBtn.visible = Characters.has_current() and not (Characters.current.get("adventure", {}) as Dictionary).is_empty()

## Resume the chosen character's saved adventure: rebuild the same world (seed +
## climate) solo and restore the dynamic state once it's populated.
func _on_continue() -> void:
	var snap: Dictionary = Characters.current.get("adventure", {})
	if snap.is_empty():
		return
	Game.player_name = String(Characters.current.get("name", "Player"))
	Game.config["mode"] = Game.Mode.ADVENTURE
	Game.config["map"] = ADVENTURE_MAP
	Game.config["seed"] = int(snap.get("seed", 0))
	Game.config["map_size"] = int(snap.get("map_size", 2))
	Game.config["mission_points"] = int(snap.get("mission_points", 10))
	Game.config["theme"] = String(snap.get("theme", ""))
	Game.config["bot_skill"] = float(snap.get("bot_skill", 1.0))
	Game.config["bot_count"] = 12
	Game.config["frag_limit"] = 0
	if String(snap.get("climate", "")) != "":
		Game.config["climate"] = String(snap["climate"])
	else:
		Game.config.erase("climate")
	Game.continue_data = snap
	if Net.host_game():
		Net.start_match()
	else:
		status_label.text = "Failed to start (port in use?)."

func _show_characters() -> void:
	setup_panel.visible = false
	create_panel.visible = false
	character_panel.visible = true
	_rebuild_char_list()

func _rebuild_char_list() -> void:
	for c in char_list.get_children():
		c.queue_free()
	if Characters.profiles.is_empty():
		var empty := Label.new()
		empty.text = "No characters yet — create one."
		empty.modulate = Color(1, 1, 1, 0.5)
		char_list.add_child(empty)
		return
	for p in Characters.profiles:
		var pd: Dictionary = p
		var b := Button.new()
		var st: Dictionary = pd.get("stats", {})
		var chosen := String(pd.get("id", "")) == String(Characters.current.get("id", ""))
		b.text = "%s%s  ·  %s  ·  %d adv, %d pts" % [
			"▶ " if chosen else "", String(pd.get("name", "?")),
			Characters.kit_name(String(pd.get("kit", "scout"))),
			int(st.get("adventures", 0)), int(st.get("points", 0))]
		b.alignment = HORIZONTAL_ALIGNMENT_LEFT
		var id := String(pd.get("id", ""))
		b.pressed.connect(func(): _choose_character(id))
		char_list.add_child(b)

func _choose_character(id: String) -> void:
	Characters.set_current(id)
	_update_char_label()
	_rebuild_char_list()

func _on_delete_character() -> void:
	if Characters.has_current():
		Characters.delete(String(Characters.current["id"]))
		_rebuild_char_list()
		_update_char_label()

func _show_create() -> void:
	%CreateName.text = ""
	%CreateBackstory.text = ""
	%CreateKit.selected = 0
	%CreateColor.color = Color(0.4, 0.6, 0.9)
	character_panel.visible = false
	create_panel.visible = true

func _on_create_confirm() -> void:
	var kit: String = Characters.KIT_IDS[clampi(%CreateKit.selected, 0, Characters.KIT_IDS.size() - 1)]
	Characters.create(%CreateName.text, %CreateColor.color, kit, %CreateBackstory.text)
	_show_characters()

func _refresh_lobby() -> void:
	if not lobby_panel.visible:
		return
	for c in lobby_players.get_children():
		c.queue_free()
	for pid in Net.players.keys():
		var l := Label.new()
		var tag := "  (host)" if pid == 1 else ""
		l.text = "• %s%s" % [Net.players[pid]["name"], tag]
		lobby_players.add_child(l)
	var summary := "%s" % Game.mode_name()
	if Game.is_coop():
		var m := Missions.get_mission(Game.config.get("mission_id", ""))
		summary += " — " + m.get("name", "?")
	elif Game.is_battle_royale():
		summary += " — last one standing"
	elif Game.is_adventure():
		summary += " — %d mission points" % int(Game.config.get("mission_points", 10))
	else:
		summary += " — frag limit %d" % int(Game.config["frag_limit"])
	summary += "\nBots: %d (%s)" % [int(Game.config["bot_count"]), _skill_name(Game.config["bot_skill"])]
	lobby_summary.text = summary

func _skill_name(v: float) -> String:
	for s in SKILLS:
		if abs(s["value"] - v) < 0.01:
			return s["name"]
	return "Custom"

# ---------------------------------------------------------------- online (central server)

## Build the account + online-play panel and slot it at the top of the setup VBox,
## above the custom-game rows. Built in code so the .tscn doesn't have to change.
func _build_online_panel() -> void:
	_setup_vbox = setup_panel.get_node("Center/Box/VBox")
	var panel := PanelContainer.new()
	panel.name = "OnlinePanel"
	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 6)
	panel.add_child(vbox)
	var header := Label.new()
	header.text = "在线对战 · Online Matchmaking"
	header.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	header.modulate = Color(1, 0.85, 0.4)
	vbox.add_child(header)
	# Auth row (visible when logged out).
	_auth_row = HBoxContainer.new()
	_auth_row.add_theme_constant_override("separation", 6)
	var al := Label.new()
	al.text = "账号"
	al.custom_minimum_size = Vector2(48, 0)
	_auth_row.add_child(al)
	_username_edit = LineEdit.new()
	_username_edit.placeholder_text = "用户名"
	_username_edit.custom_minimum_size = Vector2(140, 0)
	_auth_row.add_child(_username_edit)
	_password_edit = LineEdit.new()
	_password_edit.placeholder_text = "密码"
	_password_edit.secret = true
	_password_edit.custom_minimum_size = Vector2(140, 0)
	_password_edit.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_auth_row.add_child(_password_edit)
	_login_btn = Button.new()
	_login_btn.text = "登录"
	_auth_row.add_child(_login_btn)
	_register_btn = Button.new()
	_register_btn.text = "注册"
	_auth_row.add_child(_register_btn)
	vbox.add_child(_auth_row)
	_auth_status = Label.new()
	_auth_status.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_auth_status.modulate = Color(1, 0.6, 0.6)
	_auth_status.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	_auth_status.custom_minimum_size = Vector2(360, 0)
	vbox.add_child(_auth_status)
	# Logged-in row (visible when logged in).
	_logged_in_row = HBoxContainer.new()
	_logged_in_row.alignment = BoxContainer.ALIGNMENT_CENTER
	_logged_in_row.add_theme_constant_override("separation", 8)
	_logged_in_label = Label.new()
	_logged_in_label.custom_minimum_size = Vector2(220, 0)
	_logged_in_row.add_child(_logged_in_label)
	_logout_btn = Button.new()
	_logout_btn.text = "退出登录"
	_logged_in_row.add_child(_logout_btn)
	vbox.add_child(_logged_in_row)
	# Online buttons.
	var btn_row := HBoxContainer.new()
	btn_row.alignment = BoxContainer.ALIGNMENT_CENTER
	btn_row.add_theme_constant_override("separation", 8)
	_play_online_btn = Button.new()
	_play_online_btn.text = "在线对战"
	_play_online_btn.custom_minimum_size = Vector2(200, 0)
	btn_row.add_child(_play_online_btn)
	_history_btn = Button.new()
	_history_btn.text = "历史战绩"
	btn_row.add_child(_history_btn)
	vbox.add_child(btn_row)
	_match_status = Label.new()
	_match_status.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_match_status.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	_match_status.custom_minimum_size = Vector2(360, 0)
	_match_status.modulate = Color(0.7, 0.85, 1.0)
	vbox.add_child(_match_status)
	vbox.add_child(HSeparator.new())
	_setup_vbox.add_child(panel)
	_setup_vbox.move_child(panel, 2)   # after Title + Subtitle, above the custom rows
	_login_btn.pressed.connect(_on_login_pressed)
	_register_btn.pressed.connect(_on_register_pressed)
	_logout_btn.pressed.connect(_on_logout_pressed)
	_play_online_btn.pressed.connect(_on_play_online)
	_history_btn.pressed.connect(_on_show_history)
	_password_edit.text_submitted.connect(func(_s): _on_login_pressed())

## Full-screen overlay listing the player's recent matches (GET /api/matches).
func _build_history_panel() -> void:
	_history_panel = Control.new()
	_history_panel.name = "HistoryPanel"
	_history_panel.set_anchors_preset(Control.PRESET_FULL_RECT)
	_history_panel.mouse_filter = Control.MOUSE_FILTER_STOP
	_history_panel.visible = false
	var dim := ColorRect.new()
	dim.set_anchors_preset(Control.PRESET_FULL_RECT)
	dim.color = Color(0, 0, 0, 0.6)
	_history_panel.add_child(dim)
	var center := CenterContainer.new()
	center.set_anchors_preset(Control.PRESET_FULL_RECT)
	_history_panel.add_child(center)
	var box := PanelContainer.new()
	center.add_child(box)
	var vbox := VBoxContainer.new()
	vbox.add_theme_constant_override("separation", 8)
	vbox.custom_minimum_size = Vector2(560, 0)
	box.add_child(vbox)
	var title := Label.new()
	title.text = "历史战绩 · Recent Matches"
	title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	vbox.add_child(title)
	_history_status = Label.new()
	_history_status.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_history_status.modulate = Color(1, 1, 1, 0.6)
	vbox.add_child(_history_status)
	var scroll := ScrollContainer.new()
	scroll.custom_minimum_size = Vector2(0, 320)
	vbox.add_child(scroll)
	_history_list = VBoxContainer.new()
	_history_list.add_theme_constant_override("separation", 4)
	_history_list.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	scroll.add_child(_history_list)
	var close := Button.new()
	close.text = "关闭"
	vbox.add_child(close)
	close.pressed.connect(func(): _history_panel.visible = false)
	add_child(_history_panel)

## Toggle auth-row vs logged-in-row and the PLAY ONLINE / history buttons.
func _refresh_auth_ui() -> void:
	var logged := Auth.is_logged_in()
	_auth_row.visible = not logged
	_auth_status.text = ""
	_logged_in_row.visible = logged
	if logged:
		_logged_in_label.text = "已登录: %s" % Auth.get_username()
	_play_online_btn.disabled = not logged
	_history_btn.disabled = not logged
	if not logged:
		_queuing = false
		_pending_queue = false
		_play_online_btn.text = "在线对战"
		_match_status.text = ""

func _on_login_pressed() -> void:
	var u := _username_edit.text.strip_edges()
	var p := _password_edit.text
	if u == "" or p == "":
		_auth_status.text = "用户名和密码不能为空"
		return
	_auth_status.text = "登录中…"
	_login_btn.disabled = true
	_register_btn.disabled = true
	Auth.login(u, p)
	_password_edit.text = ""

func _on_register_pressed() -> void:
	var u := _username_edit.text.strip_edges()
	var p := _password_edit.text
	if u == "" or p == "":
		_auth_status.text = "用户名和密码不能为空"
		return
	_auth_status.text = "注册中…"
	_login_btn.disabled = true
	_register_btn.disabled = true
	Auth.register(u, p)
	_password_edit.text = ""

func _on_logout_pressed() -> void:
	if Lobby.is_lobby_connected():
		Lobby.disconnect_lobby()
	Auth.logout()

func _on_auth_logged_in(_user_id: String, _username: String) -> void:
	_login_btn.disabled = false
	_register_btn.disabled = false
	_refresh_auth_ui()
	if not Lobby.is_lobby_connected():
		Lobby.connect_lobby()

func _on_auth_logged_out() -> void:
	_refresh_auth_ui()

func _on_auth_login_failed(reason: String) -> void:
	_login_btn.disabled = false
	_register_btn.disabled = false
	var msg := reason
	match reason:
		"network": msg = "网络错误,请稍后重试"
		"credentials": msg = "用户名或密码错误"
		"taken": msg = "用户名已被占用"
		"empty": msg = "用户名和密码不能为空"
		"busy": msg = "请稍候(请求进行中)"
		_: msg = "登录失败: %s" % reason
	_auth_status.text = msg

func _on_lobby_connected() -> void:
	_match_status.text = "已连接大厅"
	_match_status.modulate = Color(0.6, 1.0, 0.6)
	if _pending_queue:
		_pending_queue = false
		_do_queue()

func _on_lobby_disconnected(reason: String) -> void:
	_queuing = false
	_pending_queue = false
	_play_online_btn.text = "在线对战"
	_match_status.text = "大厅已断开: %s" % reason
	_match_status.modulate = Color(1, 0.6, 0.6)

func _on_lobby_queued() -> void:
	_queuing = true
	_play_online_btn.text = "匹配中…(点击取消)"
	_match_status.text = "已入队,正在寻找对局…"
	_match_status.modulate = Color(0.7, 0.85, 1.0)

func _on_lobby_match_canceled() -> void:
	_queuing = false
	_play_online_btn.text = "在线对战"
	_match_status.text = "已取消匹配"

func _on_lobby_ws_error(message: String) -> void:
	_match_status.text = "错误: %s" % message
	_match_status.modulate = Color(1, 0.6, 0.6)

func _on_lobby_match_found(match_id: String, gs_host: String, gs_port: int, players: Array, mode: String) -> void:
	_queuing = false
	_play_online_btn.text = "在线对战"
	_play_online_btn.disabled = true
	_online_match = true
	# The GS identifies us by the match roster, but the ENet name registration still
	# sends Game.player_name — keep them in sync.
	Game.player_name = Auth.get_username()
	var n := players.size()
	_match_status.text = "找到对局!连接到游戏服务器 %s:%d(%d 玩家 · %s)…" % [gs_host, gs_port, n, mode]
	_match_status.modulate = Color(0.6, 1.0, 0.6)
	if not Net.join_game(gs_host, gs_port):
		_online_match = false
		_play_online_btn.disabled = not Auth.is_logged_in()
		_match_status.text = "无法启动客户端连接"
		_match_status.modulate = Color(1, 0.6, 0.6)

## PLAY ONLINE button: queue / cancel / auto-queue-once-connected.
func _on_play_online() -> void:
	if not Auth.is_logged_in():
		_auth_status.text = "请先登录"
		return
	if _online_match:
		return   # already connecting to a GS
	if _queuing:
		Lobby.cancel_match()
		return
	if not Lobby.is_lobby_connected():
		_pending_queue = true
		_match_status.text = "正在连接大厅…"
		_match_status.modulate = Color(0.7, 0.85, 1.0)
		Lobby.connect_lobby()
		return
	_do_queue()

func _do_queue() -> void:
	Lobby.queue_match(_selected_online_mode())
	_match_status.text = "正在入队…"
	_match_status.modulate = Color(0.7, 0.85, 1.0)

## Map the main mode dropdown to a backend match mode string. Adventure and Co-op
## aren't matchmade, so fall back to deathmatch for those.
func _selected_online_mode() -> String:
	match mode_option.selected:
		Game.Mode.DEATHMATCH: return "deathmatch"
		Game.Mode.TEAM_DEATHMATCH: return "team_deathmatch"
		Game.Mode.DOMINATION: return "domination"
		Game.Mode.BATTLE_ROYALE: return "battle_royale"
		_: return "deathmatch"

## BootChecker guard: in CLIENT mode (0) the menu runs normally; any other mode
## (e.g. DEDICATED_SERVER, which should have redirected already) disables online.
## The int comparison assumes the enum is ordered CLIENT=0, DEDICATED_SERVER=1 —
## adjust if subagent B defines it differently.
func _on_boot_done(mode: int) -> void:
	if mode != 0:
		_play_online_btn.disabled = true
		_match_status.text = "当前启动模式不支持在线对战"

# ---------------------------------------------------------------- match history

func _on_show_history() -> void:
	if not Auth.is_logged_in():
		_auth_status.text = "请先登录"
		return
	_history_panel.visible = true
	_history_status.text = "加载中…"
	for c in _history_list.get_children():
		c.queue_free()
	_fetch_history()

func _fetch_history() -> void:
	var url := Settings.central_url + "/api/matches"
	var headers := PackedStringArray([Auth.auth_header()])
	var err := _history_http.request(url, headers, HTTPClient.METHOD_GET)
	if err != OK:
		_history_status.text = "网络错误"
		return
	var result: Array = await _history_http.request_completed
	var rcode: int = result[0]
	var http_code: int = result[1]
	var body_bytes: PackedByteArray = result[3]
	if rcode != HTTPRequest.RESULT_SUCCESS:
		_history_status.text = "网络错误"
		return
	if http_code == 401:
		_history_status.text = "登录已过期,请重新登录"
		return
	if http_code < 200 or http_code >= 300:
		_history_status.text = "加载失败 (HTTP %d)" % http_code
		return
	var parsed = JSON.parse_string(body_bytes.get_string_from_utf8())
	var matches: Array = []
	if typeof(parsed) == TYPE_ARRAY:
		matches = parsed
	elif typeof(parsed) == TYPE_DICTIONARY and parsed.has("matches"):
		matches = parsed["matches"]
	_populate_history(matches)

func _populate_history(matches: Array) -> void:
	for c in _history_list.get_children():
		c.queue_free()
	if matches.is_empty():
		_history_status.text = "暂无对战记录"
		return
	_history_status.text = "最近 %d 场" % matches.size()
	for m in matches:
		var d: Dictionary = m
		var row := Label.new()
		var mode := String(d.get("mode", "?"))
		var status := String(d.get("status", "?"))
		var created := String(d.get("created_at", ""))
		var players = d.get("players", [])
		var pcount: int = players.size() if typeof(players) == TYPE_ARRAY else 0
		row.text = "• %s · %s · %s · %d 玩家 · id %s" % [mode, status, created, pcount, String(d.get("id", ""))]
		row.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
		row.custom_minimum_size = Vector2(520, 0)
		_history_list.add_child(row)
