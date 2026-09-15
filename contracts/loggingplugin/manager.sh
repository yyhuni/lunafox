#!/usr/bin/env bash
# The adapter lunafox_plugin_docker must propagate Docker's exit status.

lunafox_plugin_error() {
	printf 'LunaFox Loki plugin: %s\n' "$*" >&2
	return 1
}

lunafox_plugin_observe() {
	local names name row
	LUNAFOX_PLUGIN_ID='' LUNAFOX_PLUGIN_REF='' LUNAFOX_PLUGIN_ENABLED=''
	names="$(lunafox_plugin_docker plugin ls --format '{{.Name}}')" || return 1
	while IFS= read -r name; do
		case "$name" in
		lunafox-loki | lunafox-loki:latest)
			row="$(lunafox_plugin_docker plugin inspect "$name" --format '{{.Id}}|{{.PluginReference}}|{{.Enabled}}|{{range .Config.Interface.Types}}{{.}},{{end}}')" || return 1
			local interfaces
			IFS='|' read -r LUNAFOX_PLUGIN_ID LUNAFOX_PLUGIN_REF LUNAFOX_PLUGIN_ENABLED interfaces <<<"$row"
			# Docker normalizes official Hub references with the docker.io prefix.
			LUNAFOX_PLUGIN_REF="${LUNAFOX_PLUGIN_REF#docker.io/}"
			[[ "$LUNAFOX_PLUGIN_ID" =~ ^[0-9a-f]{64}$ ]] || return 1
			[[ "$LUNAFOX_PLUGIN_REF" =~ ^grafana/loki-docker-driver:[0-9]+\.[0-9]+\.[0-9]+-(amd64|arm64)$ ]] || return 1
			case ",$interfaces," in *,docker.logdriver/1.0,*) ;; *) return 1 ;; esac
			case "$LUNAFOX_PLUGIN_ENABLED" in true | false) ;; *) return 1 ;; esac
			return 0
			;;
		esac
	done <<<"$names"
}

lunafox_plugin_owned() {
	local identity
	[ -n "$LUNAFOX_PLUGIN_ID" ] || return 1
	identity="$(lunafox_plugin_docker volume inspect "lunafox_loki_plugin_$LUNAFOX_PLUGIN_ID" --format '{{index .Labels "io.lunafox.logging-plugin"}}')" || return 1
	[ "$identity" = "$LUNAFOX_PLUGIN_ID" ]
}

lunafox_plugin_record() {
	# Metadata is daemon-local and intentionally outside business volume inventory.
	lunafox_plugin_docker volume create --label "io.lunafox.logging-plugin=$LUNAFOX_PLUGIN_ID" "lunafox_loki_plugin_$LUNAFOX_PLUGIN_ID" >/dev/null || return 1
	lunafox_plugin_owned
}

lunafox_plugin_unreferenced() {
	local containers id driver
	containers="$(lunafox_plugin_docker ps -aq --no-trunc)" || return 1
	while IFS= read -r id; do
		[ -n "$id" ] || continue
		driver="$(lunafox_plugin_docker inspect --type container "$id" --format '{{.HostConfig.LogConfig.Type}}')" || return 1
		case "$driver" in
		lunafox-loki | lunafox-loki:latest | "$LUNAFOX_PLUGIN_ID")
			lunafox_plugin_error "container $id still references the plugin (including stopped containers)"
			return 1
			;;
		esac
	done <<<"$containers"
}

lunafox_plugin_prepare() {
	local mode="$1" target="$2" original_id architecture
	case "$mode" in install | start) ;; *)
		lunafox_plugin_error 'invalid preparation mode'
		return 1
		;;
	esac
	[[ "$target" =~ ^grafana/loki-docker-driver:[0-9]+\.[0-9]+\.[0-9]+-(amd64|arm64)$ ]] || {
		lunafox_plugin_error 'missing or invalid deployed reference'
		return 1
	}
	architecture="$(lunafox_plugin_docker info --format '{{.Architecture}}')" || return 1
	case "$architecture" in
	amd64 | x86_64) architecture=amd64 ;;
	arm64 | aarch64) architecture=arm64 ;;
	*)
		lunafox_plugin_error 'unsupported Docker daemon architecture'
		return 1
		;;
	esac
	[[ "$target" == *"-$architecture" ]] || {
		lunafox_plugin_error 'deployed plugin architecture differs from daemon'
		return 1
	}
	lunafox_plugin_observe || {
		lunafox_plugin_error 'cannot verify plugin identity'
		return 1
	}
	if [ -z "$LUNAFOX_PLUGIN_ID" ]; then
		lunafox_plugin_docker plugin install --alias lunafox-loki --grant-all-permissions "$target" || return 1
		lunafox_plugin_observe || return 1
		[ "$LUNAFOX_PLUGIN_REF" = "$target" ] && [ "$LUNAFOX_PLUGIN_ENABLED" = true ] || return 1
		lunafox_plugin_record || return 1
		return 0
	fi
	lunafox_plugin_owned || {
		lunafox_plugin_error 'ownership record missing or mismatched; refusing to adopt plugin'
		return 1
	}
	original_id="$LUNAFOX_PLUGIN_ID"
	if [ "$LUNAFOX_PLUGIN_REF" != "$target" ]; then
		[ "$mode" = install ] || {
			lunafox_plugin_error 'deployed version mismatch; start never upgrades plugins'
			return 1
		}
		original_id="$LUNAFOX_PLUGIN_ID"
		lunafox_plugin_unreferenced || return 1
		if [ "$LUNAFOX_PLUGIN_ENABLED" = true ]; then
			lunafox_plugin_docker plugin disable "$original_id" || return 1
		fi
		# Recheck after disabling; never force Docker past its own reference guard.
		lunafox_plugin_observe && lunafox_plugin_owned && lunafox_plugin_unreferenced || return 1
		[ "$LUNAFOX_PLUGIN_ID" = "$original_id" ] || return 1
		# Docker otherwise prompts on an explicit changed tag. Source, ownership
		# and references were checked above; this flag does not bypass usage guards.
		lunafox_plugin_docker plugin upgrade --skip-remote-check --grant-all-permissions "$original_id" "$target" || return 1
		lunafox_plugin_observe || return 1
		[ "$LUNAFOX_PLUGIN_ID" = "$original_id" ] && [ "$LUNAFOX_PLUGIN_REF" = "$target" ] || return 1
	fi
	if [ "$LUNAFOX_PLUGIN_ENABLED" = false ]; then
		lunafox_plugin_docker plugin enable "$LUNAFOX_PLUGIN_ID" || return 1
	fi
	lunafox_plugin_observe && lunafox_plugin_owned || return 1
	[ "$LUNAFOX_PLUGIN_ID" = "$original_id" ] && [ "$LUNAFOX_PLUGIN_REF" = "$target" ] && [ "$LUNAFOX_PLUGIN_ENABLED" = true ]
}

lunafox_plugin_remove() {
	local original_id
	lunafox_plugin_observe || {
		lunafox_plugin_error 'plugin retained: identity inspection failed'
		return 1
	}
	[ -n "$LUNAFOX_PLUGIN_ID" ] || return 0
	if ! lunafox_plugin_owned || ! lunafox_plugin_unreferenced; then
		lunafox_plugin_error 'plugin retained: ownership or references could not be cleared'
		return 1
	fi
	original_id="$LUNAFOX_PLUGIN_ID"
	if [ "$LUNAFOX_PLUGIN_ENABLED" = true ]; then
		lunafox_plugin_docker plugin disable "$original_id" || return 1
	fi
	lunafox_plugin_observe && lunafox_plugin_owned && lunafox_plugin_unreferenced || return 1
	[ "$LUNAFOX_PLUGIN_ID" = "$original_id" ] || return 1
	lunafox_plugin_docker plugin rm "$original_id" || return 1
	lunafox_plugin_docker volume rm "lunafox_loki_plugin_$original_id" >/dev/null
}
