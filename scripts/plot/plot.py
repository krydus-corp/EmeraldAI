from pathlib import Path
from typing import Literal, List

from PIL import Image
import matplotlib.pyplot as plt
import matplotlib.patches as patches

def plot(file: str, format: Literal["XY", "HW"], annotations: List[dict]):
    im = Image.open(file)
    size = im.size

    print(f"width: {size[0]}  height: {size[1]}")

    # Create figure and axes
    _, ax = plt.subplots()

    # Display the image
    ax.imshow(im)

    # Create a Rectangle patch
    rect: patches.Rectangle = None
    for ann in annotations:
        if format == "XY":
            left = ann["xmin"]
            top = ann["ymin"]
            width = ann["xmax"] - ann["xmin"]
            height =ann["ymax"] - ann["ymin"]
            rect = patches.Rectangle((left, top), width, height, linewidth=1, edgecolor='r', facecolor='none')
        elif format == "HW":
            rect = patches.Rectangle((ann['left'], ann['top']), ann['width'], ann['height'], linewidth=1, edgecolor='r', facecolor='none')

        ax.add_patch(rect)

    plt.show()

if __name__ == "__main__":
    # File after sent to backend and trained (train manifest)
    file = Path(__file__).parent / "8b8c68b646bf9cfa57c87a2ba455cbd7.jpeg"
    annotations = [{"left":171,"top":177,"width":138,"height":75},{"left":351,"top":178,"width":119,"height":42}]


    plot(file.as_posix(), 'HW', annotations)

    # File prior to sending to backend
    file = Path(__file__).parent / "8b8c68b646bf9cfa57c87a2ba455cbd7.jpeg"
    annotations = [{"xmax": 513, "xmin": 285,"ymax": 353,"ymin": 249}]

    plot(file.as_posix(), 'XY', annotations)
