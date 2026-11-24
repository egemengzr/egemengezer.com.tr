## Goal

Test autonomous flight algorithms against the real autopilot software without putting the aircraft at risk or going to the field.

## Architecture

```pipeline SITL: the autopilot software flies the simulated aircraft as if it were real
Simulink | flight dynamics model
Simulink Coder | embedded C code generation
Unity | physics + 3D visualisation
UDP (JSON) |
ArduPilot SITL | autopilot software
```

```pipeline HITL extension (design and docs: me · implementation: teammate)
Unity | simulation outputs
STM32 | sensor emulation
Pixhawk | real flight controller
```

![Simulink flight dynamics model](simulink.png "Simulink flight dynamics model: commands in; acceleration, angular rate, position and velocity out")

## What I did

- Converted the Simulink flight-dynamics model to C with Simulink Coder and embedded it in Unity, which runs the physics with that code and provides the 3D visualisation.
- Connected Unity to ArduPilot SITL over UDP, so the autopilot software flies the simulated aircraft as if it were real.
- Designed the architecture and wrote the technical documentation for the HITL extension: simulation outputs go to an STM32, which turns them into sensor signals for a real Pixhawk. A teammate implemented it from that document.

![SITL link and ground station](sitl.jpg "SITL link: the simulated aircraft appears in the team's ground station as if it were real")

## Status

Algorithm tests run on SITL. The HITL infrastructure is in place but could not be fully tested because the hardware was not produced in time.
