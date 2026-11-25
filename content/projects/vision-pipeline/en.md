## Goal

Run onboard target detection and tracking in real time, keep the QR-coded ground target readable in any lighting, and this season, connect all aircraft and ground systems on a shared network.

## Architecture

![Threads of the vision pipeline](threads-en.svg "Each job has its own thread. Capture writes into a single slot, so the tracker always gets the newest frame and a late frame is dropped instead of queued.")

- The camera delivers raw Bayer data. The capture thread debayers it and handles exposure and white balance in a software ISP, then drops the frame into a single slot. If the tracker slows down, no queue builds up: the newest frame wins.
- The tracker (MixFormerV2) runs on the main thread and the YOLOv8 detector on its own thread. The detector publishes its latest box and the tracker takes the target from there.
- Outputs: 720p video to the ground station over GStreamer, the target's azimuth and elevation (radians) to the flight controller as MAVLink `DEBUG_VECT`, and lock notifications to the referee server over WebSocket.

## Performance (Jun – Sep 2026)

- Converted the YOLO model and the tracker to TensorRT and moved the whole system to FP16. The two engines run on separate CUDA streams.
- The tracker rebuilt its input tensors on every frame. I replaced them with pinned buffers allocated once per role (template, online template, search), with copies going over async DMA. This stopped the periodic drops to about 40 FPS.
- Merged five separate GPU → CPU synchronisations per frame into one.
- The BGR → RGB conversion now runs on the cropped region instead of the whole frame, and an unused attention mask is no longer allocated.

```bars max=60 Frame-to-frame time measured with time.perf_counter. The camera runs at 60 FPS, so about 60 FPS is the ceiling.
Baseline (no TensorRT) | 13 | 12–14 FPS
YOLO on TensorRT (FP32) | 25 | ~25 FPS
YOLO + tracker on TensorRT (FP32) | 37.5 | 35–40 FPS
Both on FP16 | 55 | 50–60, drops to ~40
+ persistent buffers, one sync | 57.5 | stable ~55–60 FPS | *
```

## Camera and optics (Jun – Sep 2026)

- Selected the camera sensor (AR0234), lens and filter.
- The competition's QR-coded target lies on the ground on a tarp and reflects the sun around midday, a problem for many teams. After researching it, I fitted a circular polarizing (CPL) filter: light reflected from non-metallic surfaces is largely polarized, so the filter suppresses the glare. Our team never had the problem.

## Jetson environment and ground side

For the code to work in the field I also prepared the flight computer, the video coming down to the ground and the clocks.

- **Orin Nano setup script:** A staged, re-runnable script that takes the board from scratch to flight-ready. If it stops halfway, it skips finished work and resumes.
  - OpenCV is built with CUDA, cuDNN and GStreamer for this board's architecture only (`CUDA_ARCH_BIN=8.7`). Before building, the script reads `CMakeCache.txt`; if GStreamer or CUDA is not enabled, the 1.5–3 hour build never starts.
  - With 8 GB of RAM the build ran out of memory. The script adds a 16 GB swap file on the SSD and picks the number of parallel jobs as (RAM + swap) / 3 GB.
  - Vision and telemetry live in separate virtual environments, so a package update on the vision side cannot break telemetry. The script checks this by testing that neither environment can see the other's packages.
- **Video relay:** The Jetson sends a single RTP stream to the ground; the relay copies it byte for byte to QGroundControl, the observation UI (multicast) and a recording. The referee's copy is wrapped in MPEG-TS without re-encoding, because raw RTP cannot be played without a description of the stream on the receiving side. Measured latency is ~22 ms over RTP and ~105 ms over MPEG-TS.
- **Referee clock:** Lock notifications are judged against the referee server's clock. The Mac script queries the server's time API several times, takes the sample with the shortest round trip to compute the offset and sets the clock with microsecond resolution. The Orin script takes the time from the ground computer and masks the NTP services so nothing undoes the setting.

## This season: batman-adv mesh network (in progress)

I am building a single mesh network for all of the team's aircraft and ground systems. batman-adv is a layer-2 mesh protocol that runs in the Linux kernel; aircraft and ground systems share one network and can relay traffic for each other.
