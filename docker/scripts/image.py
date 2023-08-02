#!/usr/bin/env python3

# Derived from https://github.com/pimoroni/inky/blob/master/examples/7color/image.py.

import sys

from PIL import Image

from inky.auto import auto

inky = auto(ask_user=True, verbose=True)
saturation = 0.5

if len(sys.argv) == 1:
    print("""
Usage: {file} image-file [saturation]
""".format(file=sys.argv[0]))
    sys.exit(1)

image = Image.open(sys.argv[1])
resizedimage = image.resize(inky.resolution)

if len(sys.argv) > 2:
    saturation = float(sys.argv[2])

inky.set_image(resizedimage, saturation=saturation)
inky.show()
