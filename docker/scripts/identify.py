#!/usr/bin/env python3

# Derived from https://github.com/pimoroni/inky/blob/master/examples/identify.py

import sys

from inky.eeprom import read_eeprom

display = read_eeprom()

if display is None:
    print("""
No display EEPROM detected.
""")
    sys.exit(1)
else:
    print("Found: {}".format(display.get_variant()))
    print(display)
