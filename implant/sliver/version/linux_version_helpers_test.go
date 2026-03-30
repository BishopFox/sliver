package version

import "testing"

func TestParseLinuxOSRelease(t *testing.T) {
	content := "# comment\n" +
		"NAME=\"Ubuntu\"\n" +
		"VERSION=\"24.04.2 LTS (Noble Numbat)\"\n" +
		"PRETTY_NAME=\"Ubuntu 24.04.2 LTS\"\n" +
		"SPECIAL=\"quoted \\\"value\\\" \\$HOME \\`uname\\`\"\n"

	values := parseLinuxOSRelease(content)

	if got, want := values["NAME"], "Ubuntu"; got != want {
		t.Fatalf("NAME = %q, want %q", got, want)
	}
	if got, want := values["VERSION"], "24.04.2 LTS (Noble Numbat)"; got != want {
		t.Fatalf("VERSION = %q, want %q", got, want)
	}
	if got, want := values["PRETTY_NAME"], "Ubuntu 24.04.2 LTS"; got != want {
		t.Fatalf("PRETTY_NAME = %q, want %q", got, want)
	}
	if got, want := values["SPECIAL"], "quoted \"value\" $HOME `uname`"; got != want {
		t.Fatalf("SPECIAL = %q, want %q", got, want)
	}
}

func TestBuildLinuxOSRelease(t *testing.T) {
	testCases := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{
			name: "prefers name and version",
			values: map[string]string{
				"NAME":        "Ubuntu",
				"VERSION":     "24.04.2 LTS (Noble Numbat)",
				"PRETTY_NAME": "Ubuntu 24.04.2 LTS",
			},
			want: "Ubuntu 24.04.2 LTS (Noble Numbat)",
		},
		{
			name: "falls back to version id",
			values: map[string]string{
				"NAME":       "Alpine Linux",
				"VERSION_ID": "3.21.3",
			},
			want: "Alpine Linux 3.21.3",
		},
		{
			name: "falls back to pretty name",
			values: map[string]string{
				"PRETTY_NAME": "Fedora Linux 41 (Workstation Edition)",
			},
			want: "Fedora Linux 41 (Workstation Edition)",
		},
		{
			name: "falls back to name",
			values: map[string]string{
				"NAME": "Linux",
			},
			want: "Linux",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := buildLinuxOSRelease(testCase.values); got != testCase.want {
				t.Fatalf("buildLinuxOSRelease() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFormatLinuxDetailedVersion(t *testing.T) {
	version := formatLinuxDetailedVersion("Ubuntu 24.04.2 LTS (Noble Numbat)", linuxVersionInfo{
		Release: "6.8.0-57-generic",
		Version: "#59-Ubuntu SMP PREEMPT_DYNAMIC Fri Feb 14 18:29:04 UTC 2025",
		Machine: "amd64",
	})

	want := "Ubuntu 24.04.2 LTS (Noble Numbat) kernel 6.8.0-57-generic #59-Ubuntu SMP PREEMPT_DYNAMIC Fri Feb 14 18:29:04 UTC 2025 x86_64"
	if version != want {
		t.Fatalf("formatLinuxDetailedVersion() = %q, want %q", version, want)
	}
}

func TestFormatLinuxDetailedVersionFallback(t *testing.T) {
	version := formatLinuxDetailedVersion("", linuxVersionInfo{
		Sysname: "Linux",
		Release: "6.12.0",
		Machine: "i686",
	})

	want := "Linux kernel 6.12.0 x86"
	if version != want {
		t.Fatalf("formatLinuxDetailedVersion() = %q, want %q", version, want)
	}
}
