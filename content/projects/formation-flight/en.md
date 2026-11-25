## Goal

Two fixed-wing aircraft flying in formation: the leader flies its own mission while the follower holds a slot 45 m behind and 5 m above it. It was a two-person project; 80 of the 96 commits are mine.

## How it works

```pipeline
Telemetry | two aircraft, two threads
Snapshot | consistent data per cycle
Target point | 45 m behind, 5 m above
Heading and speed | PID and speed profile
GUIDED commands | speed, heading, altitude
```

- **Modes:** The leader flies its mission in AUTO, the follower is in GUIDED and receives ArduPlane's GUIDED speed, heading and altitude commands.
- **Heading:** The follower's cross-track error to the target point feeds a PID whose output is a heading correction of at most 45°.
- **Speed:** The leader's ground speed is turned into airspeed using the wind vector and an EAS→TAS ratio computed from pressure and temperature. A distance-based offset is added on top: speed up when behind, brake when close. The result stays within 14–23 m/s.
- **Lost link:** The follower keeps flying the saved heading and altitude. If the timeout runs out, it returns home with RTL.
- **Changes in the field:** The leader and follower roles can be swapped from the interface with one command. Parameters are re-read from a file every cycle, so gains can be tuned without restarting.

![Follower speed profile](speed-en.svg "The offset added to the follower's target speed against its distance behind the leader (config.param values, before the speed PID).")

## Result

In simulation the two aircraft held formation. In the field, on a very windy day, the follower could not rejoin the formation and kept waiting in place. We could not tell from that flight whether the cause was the wind or the link between the aircraft.
