## Goal

Extract the target point from a Lidar scan so that an autonomous robot can dock at its charging station with a small error. The robot is taken to be at `(0, 0)` when the scan is taken. The program finds the flat surfaces around it, picks the best docking point from their intersections and reports the robot's distance to it.

## Flow

```pipeline
TOML | URL (libcurl) or disk
Lexer + parser | hand-written
Point cloud | filter, (r, θ) → (x, y)
ANTSAC | up to 10 line segments
Intersection choice | near points, angle, distance
```

## TOML reader

I did not use a TOML library for the input file; I wrote the lexer and parser from scratch in C++.

```pipeline
File | read into memory once
Lexer | string_view tokens
Parser | table + key → route id
ranges | filter, (r, θ) → (x, y)
```

- **Zero-copy tokens.** The file is read into memory once and every token is a `string_view` into that buffer. No text is copied while reading.
- **Rule table.** The lexer looks at the next character and picks the right token generator from a table of (condition, member-function pointer) pairs. It handles integers, decimals and negative numbers, quoted strings, bare keys, brackets and braces, comments, and LF and CRLF line endings.
- **Route id.** The parser keeps the last three significant tokens (`key = value`). The active table and the key are packed into one 32-bit number (`table << 16 | key`), and a single `switch` writes the value to the right field.
- **Streamed array.** The `ranges` array is never collected into a separate list. Each value is filtered against the sensor range and turned into a Cartesian point with its angle as soon as it is read. Memory is reserved up front from the number of samples implied by the angle range.
- **Missing-field check.** Every required field that is read is marked in a bit mask, and any missing ones are reported by name at the end of the file.

## ANTSAC

RANSAC picks its sample points completely at random. In a scan with several walls close together, that can tie points from different walls into one line. In ANTSAC every point has a pheromone value and sampling is proportional to it. Points that fit a line gain pheromone, and all pheromone evaporates a little after each iteration, so good samples come forward over time.

![RANSAC and ANTSAC compared](ransac-antsac.png "A weighted RANSAC attempt (left) and ANTSAC (right) on the same scan")

The intersections of the segments found are scored on three criteria: the number of points near the intersection (~30%), the intersection angle (~35%) and the intersection's distance to both lines (~35%). The highest-scoring intersection becomes the docking point.

## My part

In this two-person project I did file download and argument parsing, the TOML lexer and parser, the ANTSAC intersection scan, the near-point calculation and the choice of the best intersection. My teammate did most of the visualisation.
