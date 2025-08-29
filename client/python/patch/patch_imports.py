import os
import re

GENERATED_ROOT = "src/suga/gen"
FROM_PATTERN = r'^from (storage|pubsub)\.v2\b'
IMPORT_PATTERN = r'^import (storage|pubsub)\.v2\b'
FROM_REPLACEMENT = r'from suga.gen.\1.v2'
IMPORT_REPLACEMENT = r'import suga.gen.\1.v2'

def patch_file(path):
    with open(path, "r", encoding="utf-8") as f:
        lines = f.readlines()

    changed = False
    out_lines = []
    for line in lines:
        # Try both patterns with re.subn
        new_line, from_count = re.subn(FROM_PATTERN, FROM_REPLACEMENT, line)
        if from_count > 0:
            line = new_line
            changed = True
        else:
            new_line, import_count = re.subn(IMPORT_PATTERN, IMPORT_REPLACEMENT, line)
            if import_count > 0:
                line = new_line
                changed = True
        out_lines.append(line)

    if changed:
        with open(path, "w", encoding="utf-8") as f:
            f.writelines(out_lines)
        print(f"Patched: {path}")

def patch_all():
    for root, _, files in os.walk(GENERATED_ROOT):
        for file in files:
            if file.endswith(".py"):
                patch_file(os.path.join(root, file))

if __name__ == "__main__":
    patch_all()
