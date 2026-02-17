#!/usr/bin/env python3
"""mine hook: * at notify stage (notify mode)

Logs all mine command invocations to a JSON Lines file for personal analytics.
This is a notify hook — it runs in the background after every command
and never blocks execution.

Install:
    cp all-commands.notify.py ~/.config/mine/hooks/
    chmod +x ~/.config/mine/hooks/all-commands.notify.py

Note: The filename uses "all-commands" as the pattern component, but rename
it to "*.notify.py" when installing to match all commands. The repo uses
"all-commands" to avoid glob issues with version control.

Test:
    mine hook test ~/.config/mine/hooks/*.notify.py
"""

import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path


def main():
    context = json.load(sys.stdin)

    data_dir = os.environ.get(
        "XDG_DATA_HOME", os.path.expanduser("~/.local/share")
    )
    log_path = Path(data_dir) / "mine" / "command_log.jsonl"
    log_path.parent.mkdir(parents=True, exist_ok=True)

    entry = {
        "command": context.get("command", "unknown"),
        "timestamp": context.get("timestamp", datetime.now(timezone.utc).isoformat()),
        "args_count": len(context.get("args", [])),
        "flags_count": len(context.get("flags", {})),
    }

    with open(log_path, "a") as f:
        f.write(json.dumps(entry) + "\n")


if __name__ == "__main__":
    main()
