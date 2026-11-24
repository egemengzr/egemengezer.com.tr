## Goal

Run onboard target detection and tracking in real time, keep the QR-coded ground target readable in any lighting, and this season, connect all vehicles on a shared network.

## Vision pipeline

```pipeline
Camera | sensor + lens + CPL filter
YOLO detection | TensorRT, FP16
Tracker | TensorRT, FP16, object reuse
Target data | to the autonomy software
```

## Performance (Jun – Sep 2026)

- Converted the YOLO model and the off-the-shelf tracker we used to TensorRT, and moved the whole system to FP16.
- Found an object in the tracker code that was rebuilt on every frame and cost a lot of time. I replaced it with a single instance created once and cleared and refilled each frame, removing the per-frame allocation and setup cost. This stopped the periodic drops to about 40 FPS.

```bars max=60 Frame-to-frame time measured with time.perf_counter. The camera runs at 60 FPS, so about 60 FPS is the ceiling.
Baseline (no TensorRT) | 13 | 12–14 FPS
YOLO on TensorRT (FP32) | 25 | ~25 FPS
YOLO + tracker on TensorRT (FP32) | 37.5 | 35–40 FPS
Both on FP16 | 55 | 50–60, drops to ~40
+ object reuse | 57.5 | stable ~55–60 FPS | *
```

## Camera and optics (Jun – Sep 2026)

- Selected the camera sensor, lens and filter.
- The competition's QR-coded target lies on the ground on a tarp and reflects the sun around midday, a problem for many teams. After researching it, I fitted a circular polarizing (CPL) filter: light reflected from non-metallic surfaces is largely polarized, so the filter suppresses the glare. Our team never had the problem.

## This season: batman-adv mesh network (in progress)

I am building a single mesh network for all of the team's air and ground vehicles. batman-adv is a layer-2 mesh protocol that runs in the Linux kernel; vehicles share one network and can relay traffic for each other.
