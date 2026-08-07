extends Node
## Boot-time mode detection.
##
## Loaded as an autoload AFTER Settings/Auth/Lobby. The main scene (main_menu.tscn)
## loads after every autoload, so _ready() here still runs before main_menu._ready()
## and can redirect a dedicated server straight into dedicated_server.tscn, skipping
## the menu (and the ComfyUI / LLM boot path the menu triggers).
##
## Mode is decided purely from environment variables, so the central server can launch
## a Game Server process with: OPENFIRE_GS_MATCH_ID=<uuid> OPENFIRE_GS_PORT=27020 ...

signal boot_done(mode: int)

enum Mode { CLIENT, DEDICATED_SERVER }

var mode: int = Mode.CLIENT

# Dedicated-server parameters (only meaningful when mode == DEDICATED_SERVER).
var gs_match_id: String = ""
var gs_port: int = 27015
var gs_mode: String = "deathmatch"
var central_url: String = ""
var internal_token: String = ""

func _ready() -> void:
	gs_match_id = OS.get_environment("OPENFIRE_GS_MATCH_ID")
	if gs_match_id != "":
		mode = Mode.DEDICATED_SERVER
		gs_port = int(OS.get_environment("OPENFIRE_GS_PORT")) if OS.get_environment("OPENFIRE_GS_PORT") != "" else 27015
		gs_mode = OS.get_environment("OPENFIRE_GS_MODE") if OS.get_environment("OPENFIRE_GS_MODE") != "" else "deathmatch"
		central_url = OS.get_environment("OPENFIRE_CENTRAL_URL") if OS.get_environment("OPENFIRE_CENTRAL_URL") != "" else Settings.central_url
		internal_token = OS.get_environment("OPENFIRE_INTERNAL_TOKEN") if OS.get_environment("OPENFIRE_INTERNAL_TOKEN") != "" else Settings.central_internal_token
		print("[BootChecker] DEDICATED_SERVER match=%s port=%d mode=%s" % [gs_match_id, gs_port, gs_mode])
		# Skip the menu entirely; dedicated_server.gd takes over from here.
		# Use call_deferred because _ready() runs while the SceneTree is still
		# building (autoloads load before the main scene); a direct change_scene
		# here triggers "Parent node is busy adding/removing children".
		get_tree().change_scene_to_file.call_deferred("res://scenes/dedicated_server.tscn")
	else:
		mode = Mode.CLIENT
		print("[BootChecker] CLIENT mode")
	boot_done.emit(mode)
