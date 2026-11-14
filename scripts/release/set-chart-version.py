# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

"""Set version and appVersion in a Helm Chart.yaml.

VERSION may include a leading v. CHART is the path to Chart.yaml.
"""

import os
from pathlib import Path

version = os.environ["VERSION"]
if version.startswith("v"):
    version = version[1:]
chart = Path(os.environ["CHART"])
lines = chart.read_text().splitlines()
saw_version = False
saw_app = False
for i, line in enumerate(lines):
    if line.startswith("version:"):
        lines[i] = f"version: {version}"
        saw_version = True
    elif line.startswith("appVersion:"):
        lines[i] = f'appVersion: "{version}"'
        saw_app = True
if not saw_version or not saw_app:
    raise SystemExit("Chart.yaml is missing version or appVersion")
chart.write_text("\n".join(lines) + "\n")
