# reMarkable Tools
Tools for working with the [reMarkable](https://remarkable.com/) notes format.

## Parser
The parser supports the v3 format for reMarkable notes.

## RM Lines Format
The `rm` format is the proprietary format used by the
[reMarkable](https://remarkable.com/) tablet. The format is used to store
drawings made on the tablet.

In *version 5*, each `.rm` file contains one page.
Each page consists of several layers which contain the *strokes* that make up
the image.

The file consists of a simple header followed by the data.
The data is structured into layers, strokes and dots.

All numeric values are **little endian**.

### Header
The header length is `43` bytes and contains ASCII encoded text:

    reMarkable .lines file, version=5

The remaining bytes are filled with whitespace.

### Layer
The header is immediately followed by `4 bytes` for a 32 bit unsigned integer
which gives us the number of layers.

The next `4 bytes` are again a 32 bit unsigned integer for the number of
*strokes* in the first layer.

### Stroke
Each stroke consists of the stroke data followed by the data for the dots.
A stroke has the following attributes:

| Size      | Datatype  | Description         |
|-----------|-----------|---------------------|
| `4 bytes` | `uint32`  | Brush Type          |
| `4 bytes` | `uint32`  | Color               |
| `4 bytes` | `uint32`  | Padding?            |
| `4 bytes` | `float32` | Brush Size          |
| `4 bytes` | -         | *unknown* (v5 only) |
| `4 bytes` | `uint32`  | Number of Dots      |

The data for the individual dots follows immediately after.

The **Brush Types** refer to the different "pencil" choices available on the
tablet. The values are different for the v3 and the v5 format.

| Version | Brush             | ID |
|---------|-------------------|----|
| v3      | Paint Brush       | 0  |
| v3      | Pencil            | 1  |
| v3      | Ballpoint         | 2  |
| v3      | Marker            | 3  |
| v3      | Fineliner         | 4  |
| v3      | Highlighter       | 5  |
| v3      | Eraser            | 6  |
| v3      | Mechanical Pencil | 7  |
| v3      | Eraser            | 8  |
|         |                   |    |
| v5      | Brush             | 12 |
| v5      | Mechanical Pencil | 13 |
| v5      | Pencil            | 14 |
| v5      | Ballpoint         | 15 |
| v5      | Marker            | 16 |
| v5      | Fineliner         | 17 |
| v5      | Highlighter       | 18 |

The **Color** is either *Black* (`0`), *Gray* (`1`) or *White* (`2`).

The **Brush Size** is the selected base size of the brush
(not to be confused with the effective width of the stroke).
Predefined sizes are *Small* (`1.875`), *Medium* (`2.0`) and *Large* (`2.125`).

### Dot
Each dot holds the following attributes:

| Size      | Datatype  | Description    |
|-----------|-----------|----------------|
| `4 bytes` | `float32` | X-Coordinate   |
| `4 bytes` | `float32` | Y-Coordinate   |
| `4 bytes` | `float32` | Speed          |
| `4 bytes` | `float32` | Tilt           |
| `4 bytes` | `float32` | Width          |
| `4 bytes` | `float32` | Pressure       |

After the last byte of the dot data is read,
The data for the next layer begins, starting with the number of strokes.

The **Coordinates** range between `0,0` and `1404,1872`,
the origin is at the top left corner.

**Speed** is a measure for how fast the stylus is drawn across the surface.
Not sure how this would affect something (maybe the density of the stroke?).

The **Pressure** values ranges from `0.0` to `1.0`.

The **Width** seems to be the effective width of the brush,
already accounted for tilt and pressure.

> Not quite sure if this is correct.
> Depending on the Brush type and size, pressure and tilt should determine
> the actual width of the stroke.

The **Tilt** value is the angle of the stylus towards the tablet surface.
It is given in radians and ranges in two intervals
from `0.0` to `1.5708` (0 to 90 degrees)
and from `4.7124` to `6.2832` (270 to 360 degrees).

---

Sources:

- https://plasma.ninja/blog/devices/remarkable/binary/format/2017/12/26/reMarkable-lines-file-format.html
- https://github.com/juruen/rmapi/
- https://www.reddit.com/r/RemarkableTablet/comments/7c5fh0/work_in_progress_format_of_the_lines_files/
