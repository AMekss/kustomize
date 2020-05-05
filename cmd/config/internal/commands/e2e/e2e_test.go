// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunE2e(t *testing.T) {
	binDir, err := ioutil.TempDir("", "kustomize-test-")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	//defer os.RemoveAll(binDir)
	build(t, binDir)

	tests := []struct {
		name           string
		args           func(string) []string
		files          func(string) map[string]string
		expectedFiles  func(string) map[string]string
		expectedErr    string
		skipIfFalseEnv string
	}{
		{
			name: "exec_function_no_args",
			args: func(d string) []string {
				return []string{
					"--enable-exec", "--exec-path", filepath.Join(d, "e2econtainerconfig"),
				}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-string-value: ''
    a-int-value: '0'
    a-bool-value: 'false'
`,
				}
			},
		},

		{
			name: "exec_function_args",
			args: func(d string) []string {
				return []string{
					"--enable-exec", "--exec-path", filepath.Join(d, "e2econtainerconfig"),
					"--", "stringValue=a", "intValue=1", "boolValue=true",
				}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-string-value: 'a'
    a-int-value: '1'
    a-bool-value: 'true'
`,
				}
			},
		},

		{
			name: "exec_function_config",
			args: func(d string) []string {
				return []string{"--enable-exec"}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: "%s"
`, filepath.Join(d, "e2econtainerconfig")),
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: "%s"
    a-string-value: ''
    a-int-value: '0'
    a-bool-value: 'false'
`, filepath.Join(d, "e2econtainerconfig"))}
			},
		},

		//
		// Starklark function tests
		//
		{
			name: "exec_function_config",
			args: func(d string) []string {
				return []string{"--enable-exec"}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"config.yaml": fmt.Sprintf(`
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: "%s"
data:
  stringValue: a
  intValue: 2
  boolValue: true
`, filepath.Join(d, "e2econtainerconfig")),
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"config.yaml": fmt.Sprintf(`
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: "%s"
    a-string-value: 'a'
    a-int-value: '2'
    a-bool-value: 'true'
data:
  stringValue: a
  intValue: 2
  boolValue: true
`, filepath.Join(d, "e2econtainerconfig")),
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-string-value: 'a'
    a-int-value: '2'
    a-bool-value: 'true'
`,
				}
			},
		},

		{
			//
			// NOTE: Do not change the expected value of this test.  It is to ensure that
			// exec functions are off by default when run from the CLI.
			// exec functions execute arbitrary code outside of a sandbox environment.
			//
			name: "exec_function_config_disabled",
			args: func(d string) []string { return []string{} },
			files: func(d string) map[string]string {
				return map[string]string{
					"config.yaml": fmt.Sprintf(`
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: "%s"
data:
  stringValue: a
  intValue: 2
  boolValue: true
`, filepath.Join(d, "e2econtainerconfig")),
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"config.yaml": fmt.Sprintf(`
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      exec:
        path: "%s"
data:
  stringValue: a
  intValue: 2
  boolValue: true
`, filepath.Join(d, "e2econtainerconfig")),
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
		},

		{
			name:        "exec_function_no_enable",
			expectedErr: "must specify --enable-exec with --exec-path",
			args: func(d string) []string {
				return []string{"--exec-path", filepath.Join(d, "e2econtainerconfig")}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-string-value: ''
    a-int-value: '0'
    a-bool-value: 'false'
`,
				}
			},
		},

		//
		// Container
		//
		{
			name:           "container_function_no_args",
			skipIfFalseEnv: "KUSTOMIZE_DOCKER_E2E",
			args: func(d string) []string {
				return []string{"--image", "gcr.io/kustomize-functions/e2econtainerconfig"}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-string-value: ''
    a-int-value: '0'
    a-bool-value: 'false'
`,
				}
			},
		},

		{
			name:           "container_function_args",
			skipIfFalseEnv: "KUSTOMIZE_DOCKER_E2E",
			args: func(d string) []string {
				return []string{
					"--image", "gcr.io/kustomize-functions/e2econtainerconfig",
					"--", "stringValue=a", "intValue=1", "boolValue=true",
				}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-string-value: 'a'
    a-int-value: '1'
    a-bool-value: 'true'
`,
				}
			},
		},

		{
			name:           "container_function_config",
			skipIfFalseEnv: "KUSTOMIZE_DOCKER_E2E",
			args:           func(d string) []string { return []string{} },
			files: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: "gcr.io/kustomize-functions/e2econtainerconfig"
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: "gcr.io/kustomize-functions/e2econtainerconfig"
    a-string-value: ''
    a-int-value: '0'
    a-bool-value: 'false'
`}
			},
		},

		{
			name:           "container_function_config",
			skipIfFalseEnv: "KUSTOMIZE_DOCKER_E2E",
			args:           func(d string) []string { return []string{} },
			files: func(d string) map[string]string {
				return map[string]string{
					"config.yaml": `
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: "gcr.io/kustomize-functions/e2econtainerconfig"
data:
  stringValue: a
  intValue: 2
  boolValue: true
`,
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"config.yaml": `
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      container:
        image: "gcr.io/kustomize-functions/e2econtainerconfig"
    a-string-value: 'a'
    a-int-value: '2'
    a-bool-value: 'true'
data:
  stringValue: a
  intValue: 2
  boolValue: true
`,
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-string-value: 'a'
    a-int-value: '2'
    a-bool-value: 'true'
`,
				}
			},
		},

		{
			name: "starlark_function_config",
			args: func(d string) []string { return []string{"--enable-star"} },
			files: func(d string) map[string]string {
				return map[string]string{
					"script.star": `
# set the foo annotation on each resource
def run(r, fc):
  for resource in r:
    resource["metadata"]["annotations"]["a-string-value"] = fc["data"]["stringValue"]
    resource["metadata"]["annotations"]["a-int-value"] = fc["data"]["intValue"]
    resource["metadata"]["annotations"]["a-bool-value"] = fc["data"]["boolValue"]

run(ctx.resource_list["items"], ctx.resource_list["functionConfig"])
`,
					"config.yaml": `
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    config.kubernetes.io/function: |
      starlark:
        path: script.star
        name: fn
data:
  boolValue: true
  intValue: 2
  stringValue: a
`,
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"config.yaml": `
apiVersion: example.com/v1alpha1
kind: Input
metadata:
  name: foo
  annotations:
    a-bool-value: true
    a-int-value: 2
    a-string-value: a
    config.kubernetes.io/function: |
      starlark:
        path: script.star
        name: fn
data:
  boolValue: true
  intValue: 2
  stringValue: a
`,
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-bool-value: true
    a-int-value: 2
    a-string-value: a
`,
				}
			},
		},

		{
			name: "starlark_function_path",
			args: func(d string) []string {
				return []string{
					"--enable-star", "--star-path", "script.star",
					"--", "stringValue=a", "intValue=2", "boolValue=true",
				}
			},
			files: func(d string) map[string]string {
				return map[string]string{
					"script.star": `
# set the foo annotation on each resource
def run(r, fc):
  for resource in r:
    resource["metadata"]["annotations"]["a-string-value"] = fc["data"]["stringValue"]
    resource["metadata"]["annotations"]["a-int-value"] = fc["data"]["intValue"]
    resource["metadata"]["annotations"]["a-bool-value"] = fc["data"]["boolValue"]

run(ctx.resource_list["items"], ctx.resource_list["functionConfig"])
`,
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
`,
				}
			},
			expectedFiles: func(d string) map[string]string {
				return map[string]string{
					"deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  annotations:
    a-bool-value: true
    a-int-value: 2
    a-string-value: a
`,
				}
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfFalseEnv != "" && os.Getenv(tt.skipIfFalseEnv) == "false" {
				t.Skip()
			}

			dir, err := ioutil.TempDir("", "kustomize-test-data-")
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer os.RemoveAll(dir)
			os.Chdir(dir)

			// write the input
			for path, data := range tt.files(binDir) {
				err := ioutil.WriteFile(path, []byte(data), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

			args := append([]string{"run", "."}, tt.args(binDir)...)
			cmd := exec.Command(filepath.Join(binDir, "kyaml"), args...)
			cmd.Dir = dir
			var stdErr, stdOut bytes.Buffer
			cmd.Stdout = &stdOut
			cmd.Stderr = &stdErr
			cmd.Env = os.Environ()

			err = cmd.Run()
			if tt.expectedErr != "" {
				if !assert.Contains(t, stdErr.String(), tt.expectedErr) {
					t.FailNow()
				}
				return
			}
			if !assert.NoError(t, err, stdErr.String()) {
				t.FailNow()
			}

			for path, data := range tt.expectedFiles(binDir) {
				b, err := ioutil.ReadFile(path)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				if !assert.Equal(t, strings.TrimSpace(data), strings.TrimSpace(string(b))) {
					t.FailNow()
				}
			}
		})
	}
}

func build(t *testing.T, binDir string) {
	build := exec.Command("go", "build", "-o",
		filepath.Join(binDir, "e2econtainerconfig"))
	build.Dir = "e2econtainerconfig"
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if !assert.NoError(t, build.Run()) {
		t.FailNow()
	}
	build = exec.Command("go", "build", "-o",
		filepath.Join(binDir, "e2econtainerflags"))
	build.Dir = "e2econtainerflags"
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if !assert.NoError(t, build.Run()) {
		t.FailNow()
	}

	build = exec.Command("go", "build", "-o", filepath.Join(binDir, "kyaml"))
	build.Dir = filepath.Join("..", "..", "..")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if !assert.NoError(t, build.Run()) {
		t.FailNow()
	}

	if os.Getenv("KUSTOMIZE_DOCKER_E2E") == "false" {
		return
	}
	build = exec.Command(
		"docker", "build", ".", "-t", "gcr.io/kustomize-functions/e2econtainerconfig")
	build.Dir = "e2econtainerconfig"
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if !assert.NoError(t, build.Run()) {
		t.FailNow()
	}
	build = exec.Command(
		"docker", "build", ".", "-t", "gcr.io/kustomize-functions/e2econtainerflags")
	build.Dir = "e2econtainerflags"
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if !assert.NoError(t, build.Run()) {
		t.FailNow()
	}
}
