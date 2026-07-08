package main

import "testing"

// TestValidateFlags locks in M2: misconfigured numeric flags produce a clear
// error instead of silent empty output.
func TestValidateFlags(t *testing.T) {
	// Snapshot the flag defaults so each subtest mutates and restores.
	cpu0, maxCpu0 := *cpu, *maxCpu
	mem0, maxMem0 := *memory, *maxMemory
	res0, lim0, cut0 := *resolution, *limit, *cutoff
	restore := func() {
		*cpu, *maxCpu = cpu0, maxCpu0
		*memory, *maxMemory = mem0, maxMem0
		*resolution, *limit, *cutoff = res0, lim0, cut0
	}
	defer restore()

	// Package defaults (mincpu=1, maxcpu=32, minmem=2, maxmem=64, resolution=7,
	// limit=20, cutoff=2) are valid.
	if err := validateFlags(); err != nil {
		t.Fatalf("default flags should be valid, got %v", err)
	}

	cases := []struct {
		name   string
		mutate func()
	}{
		{"mincpu zero", func() { *cpu = 0 }},
		{"maxcpu zero", func() { *maxCpu = 0 }},
		{"mincpu greater than maxcpu", func() { *cpu = 32; *maxCpu = 1 }},
		{"minmem zero", func() { *memory = 0 }},
		{"minmem greater than maxmem", func() { *memory = 64; *maxMemory = 2 }},
		{"resolution zero", func() { *resolution = 0 }},
		{"resolution negative", func() { *resolution = -3 }},
		{"limit zero", func() { *limit = 0 }},
		{"limit negative", func() { *limit = -5 }},
		{"cutoff negative", func() { *cutoff = -1 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			restore()
			c.mutate()
			if err := validateFlags(); err == nil {
				t.Errorf("expected validation error for %s", c.name)
			}
		})
	}

	// cutoff == 0 is ALLOWED (legitimate "highlight only free instances" filter:
	// Discount is 0 when SpotPrice is 0); only negatives are rejected.
	restore()
	*cutoff = 0
	if err := validateFlags(); err != nil {
		t.Errorf("cutoff=0 should be allowed, got %v", err)
	}
}

// TestResolveCredentials locks in H4: explicit flags win, then ALIYUN_* env,
// then ALIBABA_CLOUD_* env; whitespace-only values fall through.
func TestResolveCredentials(t *testing.T) {
	// Explicit flags take precedence over environment.
	t.Setenv("ALIYUN_ACCESS_KEY_ID", "envID")
	t.Setenv("ALIYUN_ACCESS_KEY_SECRET", "envSecret")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "officialID")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "officialSecret")

	if id, secret := resolveCredentials("flagID", "flagSecret"); id != "flagID" || secret != "flagSecret" {
		t.Errorf("flags should win, got id=%q secret=%q", id, secret)
	}

	// No flags -> ALIYUN_* env used.
	if id, secret := resolveCredentials("", ""); id != "envID" || secret != "envSecret" {
		t.Errorf("ALIYUN_* env should be used, got id=%q secret=%q", id, secret)
	}

	// ALIYUN_* blank -> official ALIBABA_CLOUD_* fallback.
	t.Setenv("ALIYUN_ACCESS_KEY_ID", "")
	t.Setenv("ALIYUN_ACCESS_KEY_SECRET", "")
	if id, secret := resolveCredentials("", ""); id != "officialID" || secret != "officialSecret" {
		t.Errorf("ALIBABA_CLOUD_* should be the fallback, got id=%q secret=%q", id, secret)
	}

	// Whitespace-only flag falls through to env.
	if id, _ := resolveCredentials("   ", "   "); id != "officialID" {
		t.Errorf("whitespace-only flag should fall through to env, got id=%q", id)
	}

	// Nothing set -> empty.
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_ID", "")
	t.Setenv("ALIBABA_CLOUD_ACCESS_KEY_SECRET", "")
	if id, secret := resolveCredentials("", ""); id != "" || secret != "" {
		t.Errorf("expected empty credentials when nothing is set, got id=%q secret=%q", id, secret)
	}
}
