// Copyright (C) 2025-2026 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

package main

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

const (
	chartDir     = "dist/chart/"
	templatesDir = chartDir + "templates/"

	chartPath       = chartDir + "Chart.yaml"
	chartValuesPath = chartDir + "values.yaml"

	managerPath       = templatesDir + "manager/manager.yaml"
	downloaderPath    = templatesDir + "extras/chalkmark-extract.yaml"
	uploaderPath      = templatesDir + "extras/results.yaml"
	reportServicePath = templatesDir + "extras/report-http.yaml"
)

// replacements is the list of regexp replacements
// to apply per file.
// be sure to use $$ to escape the $ character in
// replacement text
var replacements = map[string][]replacement{
	managerPath: {
		{
			Pattern: regexp.MustCompile(`(?m)^([ ]+)labels:`),
			Replacement: "${1}labels:\n" +
				"${1}  {{- range $$key, $$val := .Values.manager.labels }}\n" +
				"${1}  {{ $$key }}: {{ $$val | quote }}\n" +
				"${1}  {{- end}}",
		},
		{
			Pattern: regexp.MustCompile(`(?m)^([ ]+)annotations:`),
			Replacement: "${1}annotations:\n" +
				"${1}  {{- range $$key, $$val := .Values.manager.annotations }}\n" +
				"${1}  {{ $$key }}: {{ $$val | quote }}\n" +
				"${1}  {{- end}}",
		},
		{
			Pattern: regexp.MustCompile(`(?m)^([ ]+)(- args:)`),
			Replacement: "${1}- args:\n" +
				// SQS args
				"${1}  {{- if .Values.intake.sqs.enable}}\n" +
				"${1}  - --sqs-queue-url={{ .Values.intake.sqs.queue_url }}\n" +
				"${1}  {{- end }}\n" +
				// HTTP args
				"${1}  {{- if .Values.intake.http.enable}}\n" +
				"${1}  - --report-http-bind-address=:{{ .Values.intake.http.port }}\n" +
				"${1}  - --report-http-secure={{ .Values.intake.http.secure }}\n" +
				"${1}  {{- if .Values.intake.http.secure }}\n" +
				"${1}  - --report-http-cert-path={{ .Values.intake.http.cert.path }}\n" +
				"${1}  - --report-http-cert-name={{ .Values.intake.http.cert.name }}\n" +
				"${1}  - --report-http-cert-key={{ .Values.intake.http.cert.key }}\n" +
				"${1}  {{- end }}\n" +
				"${1}  {{- end }}",
		},
		{
			Pattern: regexp.MustCompile(`(?m)^([ ]+)env: (\[\])?`),
			Replacement: "${1}env:\n" +
				"${1}  {{- with .Values.manager.env }}\n" +
				"${1}  {{- toYaml . | nindent 10 }}\n" +
				"${1}  {{- end}}",
		},
		{
			Pattern: regexp.MustCompile(`(?m)^([ ]+)volumeMounts:`),
			Replacement: "${1}volumeMounts:\n" +
				"${1}  {{- with .Values.manager.volumeMounts }}\n" +
				"${1}  {{- toYaml . | nindent 10}}\n" +
				"${1}  {{- end}}",
		},
		{
			Pattern: regexp.MustCompile(`(?m)^([ ]+)volumes:`),
			Replacement: "${1}volumes:\n" +
				"${1}  {{- with .Values.manager.volumes }}\n" +
				"${1}  {{- toYaml . | nindent 8}}\n" +
				"${1}  {{- end}}",
		},
	},
	downloaderPath: {
		{
			Pattern:     regexp.MustCompile(`(?m)^([ ]+)image:.*$`),
			Replacement: "${1}image: {{ .Values.downloader.image.repository }}:{{ .Values.downloader.image.tag }}",
		},
		{
			Pattern:     regexp.MustCompile(`(?m)^([ ]+)imagePullPolicy:.*$`),
			Replacement: "${1}imagePullPolicy: {{ .Values.downloader.image.pullPolicy }}",
		},
	},
	uploaderPath: {
		{
			Pattern:     regexp.MustCompile(`(?m)^([ ]+)image:.*$`),
			Replacement: "${1}image: {{ .Values.uploader.image.repository }}:{{ .Values.uploader.image.tag }}",
		},
		{
			Pattern:     regexp.MustCompile(`(?m)^([ ]+)imagePullPolicy:.*$`),
			Replacement: "${1}imagePullPolicy: {{ .Values.uploader.image.pullPolicy }}",
		},
	},
	reportServicePath: {
		{
			Pattern:     regexp.MustCompile(`^`),
			Replacement: "{{- if .Values.intake.http.enable }}\n",
		},
		{
			Pattern:     regexp.MustCompile(`$`),
			Replacement: "\n{{- end}}",
		},
		{
			Pattern:     regexp.MustCompile(`(?m)^([ ]+)port:.*$`),
			Replacement: "${1}port: {{ .Values.intake.http.port }}",
		},
		{
			Pattern:     regexp.MustCompile(`(?m)^([ ]+)targetPort:.*$`),
			Replacement: "${1}targetPort: {{ .Values.intake.http.port }}",
		},
	},
}

//go:embed Chart.yaml.template
var chartTmpl string

func patchHelmChart(req *external.PluginRequest) (external.PluginResponse, error) {
	resp := external.PluginResponse{
		APIVersion: req.APIVersion,
		Command:    req.Command,
		Universe:   make(map[string]string),
	}

	if err := applyReplacements(req, resp.Universe); err != nil {
		return resp, err
	}

	// for now we just YQ since it preserves comments
	// eventually we should switch to a comment preserving way to set
	// values.yaml
	// mergedValues, err := mergeYAML(req.Universe[chartValuesPath], valuesYAML)
	// if err != nil {
	// return resp, err
	// }
	// resp.Universe[chartValuesPath] = mergedValues

	chart, err := templateContent("Chart.yaml", chartTmpl, map[string]string{
		"ChartVersion": strings.TrimLeft(os.Getenv("CHALKULAR_HELM_VERSION"), "v"),
		"AppVersion":   strings.TrimLeft(os.Getenv("CHALKULAR_VERSION"), "v"),
	})
	if err != nil {
		return resp, err
	}

	resp.Universe[chartPath] = chart
	return resp, nil
}

type replacement struct {
	Pattern     *regexp.Regexp
	Replacement string
}

func (r replacement) apply(content string) (string, error) {
	if !r.Pattern.MatchString(content) {
		return content, fmt.Errorf("patch %q: pattern did not match", r.Pattern.String())
	}
	return r.Pattern.ReplaceAllString(content, r.Replacement), nil
}

func applyReplacements(req *external.PluginRequest, u map[string]string) error {
	for path, rs := range replacements {
		content, ok := req.Universe[path]
		if !ok {
			return fmt.Errorf("file not found in universe: %s", u[path])
		}
		for _, replace := range rs {
			var err error
			content, err = replace.apply(content)
			if err != nil {
				return err
			}
		}
		u[path] = content
	}
	return nil
}

func templateContent(name, content string, values map[string]string) (string, error) {
	t, err := template.New(name).Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}
	builder := &strings.Builder{}
	err = t.Execute(builder, values)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return builder.String(), nil
}
