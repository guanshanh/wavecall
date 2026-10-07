import { useState, useEffect, useCallback } from "react";

interface AudioDevice {
  deviceId: string;
  label: string;
}

/**
 * useAudio — manages audio device enumeration, selection, and volume detection.
 */
export function useAudio() {
  const [devices, setDevices] = useState<AudioDevice[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>("");
  const [volume, setVolume] = useState(0);

  // Enumerate audio input devices
  useEffect(() => {
    async function enumerate() {
      try {
        // Need a temporary stream to get device labels
        const tempStream = await navigator.mediaDevices.getUserMedia({ audio: true });
        const deviceList = await navigator.mediaDevices.enumerateDevices();
        const audioInputs = deviceList
          .filter((d) => d.kind === "audioinput")
          .map((d) => ({ deviceId: d.deviceId, label: d.label || "未知麦克风" }));
        setDevices(audioInputs);
        if (audioInputs.length > 0 && !selectedDeviceId) {
          setSelectedDeviceId(audioInputs[0]!.deviceId);
        }
        // Release the temporary stream
        tempStream.getTracks().forEach((t) => t.stop());
      } catch (err) {
        console.error("Failed to enumerate audio devices:", err);
      }
    }
    enumerate();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Select a specific device
  const selectDevice = useCallback((deviceId: string) => {
    setSelectedDeviceId(deviceId);
  }, []);

  // Analyze volume from a MediaStream (for speaking indicator)
  const analyzeVolume = useCallback((stream: MediaStream) => {
    const audioContext = new AudioContext();
    const source = audioContext.createMediaStreamSource(stream);
    const analyser = audioContext.createAnalyser();
    analyser.fftSize = 256;
    source.connect(analyser);

    const dataArray = new Uint8Array(analyser.frequencyBinCount);

    const check = () => {
      analyser.getByteFrequencyData(dataArray);
      const avg = dataArray.reduce((sum, v) => sum + v, 0) / dataArray.length;
      setVolume(avg / 255); // normalize to 0-1
      requestAnimationFrame(check);
    };
    check();

    return () => {
      audioContext.close();
    };
  }, []);

  return {
    devices,
    selectedDeviceId,
    selectDevice,
    volume,
    analyzeVolume,
  };
}
