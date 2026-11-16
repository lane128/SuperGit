"""Sample an owned native process and descendants; RSS may count shared pages twice."""
import copy
import subprocess
import threading


class NativeMemorySampler:
    def __init__(self, pid):
        self.pid = pid
        self._phase = "launch"
        self._stop = threading.Event()
        self._lock = threading.Lock()
        self._data = {"app_peak_sampled_rss_kib": 0,
                      "app_and_descendants_peak_sampled_rss_kib": 0,
                      "rss_samples": 0, "rss_by_phase": {}}
        self._thread = threading.Thread(target=self._sample, daemon=True)
        self._thread.start()

    def phase(self, name):
        with self._lock:
            self._phase = name

    def result(self):
        with self._lock:
            return copy.deepcopy(self._data)

    def close(self):
        self._stop.set()
        self._thread.join(timeout=4)
        if self._thread.is_alive():
            raise RuntimeError("RSS sampler did not stop")

    def _sample(self):
        while not self._stop.is_set():
            try:
                output = subprocess.check_output(["ps", "-axo", "pid=,ppid=,rss="], text=True, timeout=3)
                rows = [tuple(map(int, line.split())) for line in output.splitlines() if line.strip()]
                descendants = {self.pid}
                while True:
                    expanded = descendants | {pid for pid, parent, rss in rows if parent in descendants}
                    if expanded == descendants:
                        break
                    descendants = expanded
                app_rss = sum(rss for pid, parent, rss in rows if pid == self.pid)
                total = sum(rss for pid, parent, rss in rows if pid in descendants)
                with self._lock:
                    data = self._data
                    data["app_peak_sampled_rss_kib"] = max(data["app_peak_sampled_rss_kib"], app_rss)
                    data["app_and_descendants_peak_sampled_rss_kib"] = max(data["app_and_descendants_peak_sampled_rss_kib"], total)
                    data["rss_samples"] += 1
                    phase = data["rss_by_phase"].setdefault(self._phase, {"app_peak_rss_kib": 0, "total_peak_rss_kib": 0, "samples": 0})
                    phase["app_peak_rss_kib"] = max(phase["app_peak_rss_kib"], app_rss)
                    phase["total_peak_rss_kib"] = max(phase["total_peak_rss_kib"], total)
                    phase["samples"] += 1
            except (subprocess.SubprocessError, ValueError) as error:
                with self._lock:
                    self._data.setdefault("rss_sampling_errors", []).append(str(error))
            self._stop.wait(0.1)
