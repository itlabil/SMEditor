package whisper

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
	}{
		{"simple version", "whisper.cpp 1.6.0\n", "whisper.cpp 1.6.0"},
		{"with extra lines", "whisper.cpp 1.6.0\nusage: whisper-cli [options]\n", "whisper.cpp 1.6.0"},
		{"empty output", "", ""},
		{"cpu build, backend line first", string(readTestdata(t, "version_cpu.txt")), "whisper.cpp version: 1.9.4"},
		{"cuda build, device lines first", string(readTestdata(t, "version_cuda.txt")), "whisper.cpp version: 1.9.4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseVersion(tc.output); got != tc.want {
				t.Errorf("parseVersion(%q) = %q, want %q", tc.output, got, tc.want)
			}
		})
	}
}

func TestParseBackend(t *testing.T) {
	cases := []struct {
		name        string
		output      string
		wantBackend string
		wantGPU     string
	}{
		{"cuda build --version", string(readTestdata(t, "version_cuda.txt")), BackendGPU, "NVIDIA GeForce GTX 1650 with Max-Q Design"},
		{"cpu-only build --version", string(readTestdata(t, "version_cpu.txt")), BackendCPU, ""},
		{
			"ggml-cuda.dll failed to load, silent cpu fallback",
			`load_backend: loaded CPU backend from D:\SMEditor\tools\whisper\ggml-cpu-haswell.dll` + "\nwhisper.cpp version: 1.9.4\n",
			BackendCPU, "",
		},
		{"model load on gpu", "whisper_backend_init_gpu: using CUDA0 backend\n", BackendGPU, ""},
		{
			"cuda loaded but no GPU at model load",
			`load_backend: loaded CUDA backend from D:\SMEditor\tools\whisper\ggml-cuda.dll` + "\nwhisper_backend_init_gpu: no GPU found\n",
			BackendCPU, "",
		},
		{"old build without backend lines", "whisper.cpp 1.6.0\n", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend, gpu := parseBackend(tc.output)
			if backend != tc.wantBackend || gpu != tc.wantGPU {
				t.Errorf("parseBackend() = (%q, %q), want (%q, %q)", backend, gpu, tc.wantBackend, tc.wantGPU)
			}
		})
	}
}
