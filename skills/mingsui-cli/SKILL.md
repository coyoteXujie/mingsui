---
name: mingsui-cli
description: Use MingSui CLI when an AI agent needs to inspect or configure local proxy access, import MingSui or airport subscriptions, run commands through the local HTTP/SOCKS proxy, export Mihomo configs, or diagnose why command-line network access is not working. Trigger for tasks mentioning mingsui, 明隧, proxy env vars, npm-installed mingsui CLI, AI tools needing external network access, or running curl/npm/git/Claude/Codex through MingSui.
---

# MingSui CLI

Use `mingsui` as the command-line control surface for MingSui. Prefer command-scoped proxy setup for AI agents so the task does not depend on global desktop proxy state.

## First Checks

Run these before changing configuration:

```bash
mingsui version
mingsui status
mingsui config path
```

Use JSON when another tool or script will parse the result:

```bash
mingsui status -json
mingsui doctor -json
```

Do not print full subscription URLs, tokens, proxy URLs, or exported configs containing secrets in final answers. Summarize counts, selected node names, and error messages instead.

## Proxy Scope

Assume `mingsui connect` starts local proxy listeners only. It does not modify the parent shell, system proxy, browser settings, or TUN/virtual network adapter by itself.

Use one of these patterns for AI commands:

```bash
mingsui exec -- curl https://example.com
```

If no separate `mingsui connect` process is running, prefer the self-contained form. It starts Mihomo for airport nodes and starts the MingSui client for relay profiles:

```bash
mingsui exec -connect -- curl https://example.com
```

or:

```bash
eval "$(mingsui env)"
curl https://example.com
```

`mingsui exec` affects only the child command. `eval "$(mingsui env)"` affects only the current shell and processes started from it. Already-running AI agents, browsers, and desktop apps will not inherit these variables retroactively.

For browser or desktop app traffic on supported Linux/GNOME environments, use:

```bash
mingsui system-proxy enable
mingsui system-proxy status
mingsui system-proxy disable
```

Do not assume system proxy support exists on every OS or desktop environment; report unsupported status clearly.

If local proxy authentication is enabled, do not use `mingsui system-proxy enable`; use `mingsui exec` or configure browser proxy authentication manually.

## Connection Workflow

For a MingSui relay profile:

```bash
mingsui config init -relay <host:port> -token <token>
mingsui doctor
mingsui exec -connect -- curl https://example.com
```

Use `mingsui connect` when a long-running local proxy is needed. Keep it running while using the proxy. In another terminal or subprocess, run commands through `mingsui exec` or after `eval "$(mingsui env)"`.

For a subscription import:

```bash
mingsui import -source <file-or-url>
mingsui import -source <subscription-url> -subscription airport
mingsui status
mingsui config proxy list
mingsui config proxy select <node-name>
mingsui config subscription add airport -url <subscription-url>
mingsui config subscription sync airport
```

If the active profile is an airport node and direct connect reports that the general proxy kernel is not connected yet, export a Mihomo config for manual kernel testing:

```bash
mingsui kernel export -output /tmp/mingsui-mihomo.yaml
```

Treat `/tmp/mingsui-mihomo.yaml` as sensitive because it can contain node passwords or tokens.

## Browser And System Proxy

Do not claim that the CLI automatically makes the browser use MingSui unless `mingsui system-proxy enable` has succeeded. For browser traffic, the user must configure the browser or system proxy to the local listeners while `mingsui connect` or a compatible kernel is running:

```text
HTTP proxy:   127.0.0.1:18081
SOCKS5 proxy: 127.0.0.1:18080
```

TUN generally requires elevated permissions or a kernel backend such as Mihomo configured with TUN support.

## Troubleshooting

If a command cannot access the network:

1. Check `mingsui status -json`.
2. Check whether the connection process is actually running.
3. Confirm the command is launched with `mingsui exec -- ...` or from a shell where `eval "$(mingsui env)"` was run.
4. For browser traffic, check `mingsui system-proxy status`.
5. Use `mingsui doctor` to diagnose the active mode. It checks Mihomo for airport nodes and relay health for relay profiles.
6. Avoid exposing private tokens or full subscription URLs while reporting the issue.
