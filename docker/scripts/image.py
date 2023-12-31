#!/usr/bin/env python3

# Derived from https://github.com/pimoroni/inky/blob/master/examples/7color/image.py.

import sys, os

from PIL import Image, ImageOps

from inky.auto import auto

inky = auto(ask_user=True, verbose=True)
saturation = 0.5

if len(sys.argv) != 5:
    print("""
Usage: {file} image-file [rotation] [saturation] [fitType]
""".format(file=sys.argv[0]))
    sys.exit(1)

imgFullPath = sys.argv[1]
image = Image.open(imgFullPath)

image.save(os.path.join(os.path.dirname(imgFullPath), "debug_initial.jpg"))

image = image.rotate(float(sys.argv[2]), expand=True)
image.save(os.path.join(os.path.dirname(imgFullPath), "debug_rotated.jpg"))

saturation = float(sys.argv[3])

if sys.argv[4] == "padToFit":
    image = ImageOps.pad(image, inky.resolution)
else:
    image = ImageOps.fit(image, inky.resolution)

image.save(os.path.join(os.path.dirname(imgFullPath), "debug_fitted.jpg"))


inky.set_image(image, saturation=saturation)
inky.show()
