import re
import time
import serial
import serial.tools.list_ports
from dataclasses import dataclass
from typing import Optional, List, Tuple


class SerialPort:
    def __init__(self, port: str, baudrate: int = 115200, timeout: float = 1.0):
        self.port = port
        self.baudrate = baudrate
        self.timeout = timeout
        self.ser = serial.Serial(port, baudrate=baudrate, timeout=timeout)

    def send(self, command: str):
        if not command.endswith('\n'):
            command += '\n'
        self.ser.write(command.encode('utf-8'))

    def receive(self) -> str:
        return self.ser.readline().decode('utf-8').strip()

    def close(self):
        self.ser.close()

    @staticmethod
    def enumerate():
        rp2040_devices = []
        for port in serial.tools.list_ports.comports():
            if (
                "RP2" in port.description
                or "Pico" in port.description
                or "Board" in port.description
                or "MicroPython" in port.description
                or (port.vid, port.pid) in [
                    (0x2E8A, 0x000a),
                    (0x2E8A, 0x0005),
                    (0x2E8A, 0x0003),
                ]
            ):
                rp2040_devices.append((port.device, port.description))

        return rp2040_devices


def parse_go_duration(duration_str: str) -> float:
    """
    Parse Go-style time.Duration strings into seconds as float.
    """
    time_units = {
        'ns': 1e-9,
        'us': 1e-6,
        'µs': 1e-6,
        'ms': 1e-3,
        's': 1.0,
        'm': 60.0,
        'h': 3600.0
    }
    pattern = re.compile(r'(?P<value>[\d.]+)(?P<unit>ns|us|µs|ms|s|m|h)')
    matches = pattern.findall(duration_str)

    if not matches:
        raise ValueError(f"Invalid Go duration format: {duration_str}")

    total_seconds = 0.0
    for value, unit in matches:
        total_seconds += float(value) * time_units[unit]
    return total_seconds


def format_go_duration(seconds: float) -> str:
    """
    Convert a float in seconds to a Go-style time.Duration string.
    """
    if seconds < 10e-3:
        return f"{int(seconds * 1e6)}us"
    elif seconds < 1.0:
        return f"{int(seconds * 1e3)}ms"
    else:
        return f"{seconds:.3f}s".rstrip('0').rstrip('.')


@dataclass
class PulseTrainItem:
    delay: float        # in seconds
    pulse_width: float  # in seconds


@dataclass
class StateResult:
    stages: int
    voltageA: float  # in volts
    voltageB: float  # in volts
    senseA: bool     # pin state
    senseB: bool     # pin state
    chronoA: bool    # pin state
    chronoB: bool    # pin state


@dataclass
class DriveResult:
    count: int    # number of the shot
    dA: float     # dwell time in stage 1
    dB: float     # dwell time in stage 2
    interStage: float # duration between stages


@dataclass
class ShotResult:
    count: int       # number of registered shots
    valid: bool      # whether the measurement is valid
    speed: float     # registered speed
    dA: float        # dwell time in sensor 1
    dB: float        # dwell time in sensor 2


@dataclass
class DriveCommandResult:
    drive: Optional[DriveResult] = None
    shot: Optional[ShotResult] = None
    error: Optional[str] = None
    drive_raw: Optional[str] = None
    shot_raw: Optional[str] = None


class RemadrDevice(SerialPort):
    @property
    def state(self) -> StateResult:
        self.send("?")
        line = self.receive()
        if line.startswith("STATE"):
            parts = line.split()
            return StateResult(
                stages=int(parts[1]),
                voltageA=float(parts[2]),
                voltageB=float(parts[3]),
                senseA=parts[4]=="true",
                senseB=parts[5]=="true",
                chronoA=parts[6]=="true",
                chronoB=parts[7]=="true",
            )
        raise RuntimeError(f"Unexpected response: {line}")

    def set_stage(self, index: int, pulse_train: list[PulseTrainItem]) -> list[PulseTrainItem]:
        """
        Set timing shape for a stage using a list of PulseTrainItem(delay, pulse_width).
        Delays and pulse widths are in seconds and converted to Go-style durations.
        """
        durations = []
        for item in pulse_train:
            durations.append(format_go_duration(item.delay))
            durations.append(format_go_duration(item.pulse_width))

        cmd = f"s {index}," + ",".join(durations)
        self.send(cmd)
        response = self.receive()
        if response.startswith("STAGE !"):
            raise RuntimeError(response)
        
        parts = response.split()[2:]
        response = [PulseTrainItem(parse_go_duration(delay), parse_go_duration(width)) for delay, width in zip(parts[::2], parts[1::2])]

        return response

    def test(self, data: str = "") -> str:
        self.send(f"t {data}")
        return self.receive()

    def drive(self, timeout: float = 2.0) -> DriveCommandResult:
        self.send("d")
        deadline = time.time() + timeout
        result = DriveCommandResult()

        while time.time() < deadline:
            line = self.receive()
            if not line:
                continue

            if line.startswith("DRIVE !"):
                result.error = line
                break

            elif line.startswith("DRIVE"):
                parts = line.split()
                if len(parts) == 5:
                    result.drive = DriveResult(
                        count=int(parts[1]),
                        dA=parse_go_duration(parts[2]),
                        dB=parse_go_duration(parts[3]),
                        interStage=parse_go_duration(parts[4]),
                    )
                result.drive_raw = line

            elif line.startswith("SHOT"):
                parts = line.split()
                if len(parts) == 5:
                    result.shot = ShotResult(
                        count=int(parts[1]),
                        valid=True,
                        speed=float(parts[2]),
                        dA=parse_go_duration(parts[3]),
                        dB=parse_go_duration(parts[4]),
                    )
                result.shot_raw = line

            if result.drive and result.shot:
                break

        if not result.drive and not result.shot and not result.error:
            raise TimeoutError("No response received after drive command.")
        return result
