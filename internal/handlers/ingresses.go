/*
Copyright (c) 2025 Kubotal.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	_ "embed"
	"html/template"
	"net/http"
	"sort"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const linkAnnPrefix = "kindex.kubotal.io/link"

//go:embed ingresses.html
var ingressesHTML string

// IngressLink is one row on the index page.
type IngressLink struct {
	Display     string
	Target      string
	Description string
}

// IngressesPageData is passed to the HTML template.
type IngressesPageData struct {
	Links []IngressLink
	Error string
}

// IngressesHandler lists cluster ingresses and renders link entries.
func IngressesHandler(client kubernetes.Interface) http.Handler {
	tpl := template.Must(template.New("ingresses").Parse(ingressesHTML))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		list, err := client.NetworkingV1().Ingresses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
		data := IngressesPageData{}
		if err != nil {
			data.Error = err.Error()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_ = tpl.Execute(w, data)
			return
		}
		var links []IngressLink
		for i := range list.Items {
			links = append(links, linksForIngress(&list.Items[i])...)
		}
		sort.Slice(links, func(i, j int) bool {
			return strings.ToLower(links[i].Display) < strings.ToLower(links[j].Display)
		})
		data.Links = links
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tpl.Execute(w, data)
	})
}

func isLinkAnnotationKey(k string) bool {
	if k == linkAnnPrefix {
		return true
	}
	if !strings.HasPrefix(k, linkAnnPrefix) {
		return false
	}
	if len(k) == len(linkAnnPrefix) {
		return true
	}
	switch k[len(linkAnnPrefix)] {
	case '.', '/', '-':
		return true
	default:
		return false
	}
}

func linkAnnotations(ing *networkingv1.Ingress) map[string]string {
	out := make(map[string]string)
	for k, v := range ing.Annotations {
		if isLinkAnnotationKey(k) {
			out[k] = v
		}
	}
	return out
}

func firstIngressHost(ing *networkingv1.Ingress) string {
	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			return rule.Host
		}
	}
	return ""
}

func displayFromHost(host string) string {
	i := strings.IndexByte(host, '.')
	if i <= 0 {
		return host
	}
	return host[:i]
}

func hostInTLSSpec(ing *networkingv1.Ingress, host string) bool {
	for _, t := range ing.Spec.TLS {
		for _, h := range t.Hosts {
			if h == host {
				return true
			}
		}
	}
	return false
}

func sslPassthrough(ing *networkingv1.Ingress) bool {
	if ing.Annotations == nil {
		return false
	}
	for _, key := range []string{
		"nginx.ingress.kubernetes.io/ssl-passthrough",
		"ingress.kubernetes.io/ssl-passthrough",
	} {
		if strings.EqualFold(ing.Annotations[key], "true") {
			return true
		}
	}
	return false
}

func schemeForHost(ing *networkingv1.Ingress, host string) string {
	if hostInTLSSpec(ing, host) || sslPassthrough(ing) {
		return "https"
	}
	return "http"
}

func joinHostPath(host, path string) string {
	if path == "" {
		return host
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return host + path
}

func buildTarget(ing *networkingv1.Ingress, host, path string) string {
	s := schemeForHost(ing, host)
	return s + "://" + joinHostPath(host, path)
}

func linksForIngress(ing *networkingv1.Ingress) []IngressLink {
	host := firstIngressHost(ing)
	ann := linkAnnotations(ing)

	if len(ann) == 1 {
		var only string
		for _, v := range ann {
			only = v
		}
		if only == "" {
			return nil
		}
		parts := strings.SplitN(only, ":", 3)
		if parts[0] == "" {
			return nil
		}
	}

	if len(ann) > 0 {
		keys := make([]string, 0, len(ann))
		for k := range ann {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var out []IngressLink
		for _, k := range keys {
			v := ann[k]
			if v == "" {
				continue
			}
			parts := strings.SplitN(v, ":", 3)
			display := parts[0]
			if display == "" {
				continue
			}
			var path, desc string
			if len(parts) > 1 {
				path = parts[1]
			}
			if len(parts) > 2 {
				desc = parts[2]
			}
			if host == "" {
				continue
			}
			out = append(out, IngressLink{
				Display:     display,
				Target:      buildTarget(ing, host, path),
				Description: desc,
			})
		}
		return out
	}

	if host == "" {
		return nil
	}
	return []IngressLink{{
		Display: displayFromHost(host),
		Target:  buildTarget(ing, host, ""),
	}}
}
