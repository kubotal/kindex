/*
Copyright (c) 2026 Kubotal.

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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayversioned "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"kindex/internal/global"
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
	ClusterName string
	Version     string
	Mode        string // "dark" or "light"
	Links       []IngressLink
	Error       string
}

// IngressesHandler lists cluster Ingresses and Gateway API HTTPRoutes / TLSRoutes, then renders link entries.
// mode must be "dark" or "light" (validated by the caller).
// gw may be nil if the Gateway API client could not be constructed; Gateway routes are skipped in that case.
func IngressesHandler(client kubernetes.Interface, gw gatewayversioned.Interface, clusterName, mode string) http.Handler {
	tpl := template.Must(template.New("ingresses").Parse(ingressesHTML))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		data := IngressesPageData{ClusterName: clusterName, Version: global.Version, Mode: mode}

		list, err := client.NetworkingV1().Ingresses(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
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

		if gw != nil {
			gatewaysByKey := make(map[string]*gatewayv1.Gateway)
			if gwl, err := gw.GatewayV1().Gateways(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
				for i := range gwl.Items {
					k := gwl.Items[i].Namespace + "/" + gwl.Items[i].Name
					gatewaysByKey[k] = &gwl.Items[i]
				}
			} else if !isGatewayAPIMissing(err) {
				data.Error = err.Error()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = tpl.Execute(w, data)
				return
			}

			if hr, err := gw.GatewayV1().HTTPRoutes(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
				for i := range hr.Items {
					links = append(links, linksForHTTPRoute(&hr.Items[i], gatewaysByKey)...)
				}
			} else if !isGatewayAPIMissing(err) {
				data.Error = err.Error()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = tpl.Execute(w, data)
				return
			}

			if tr, err := gw.GatewayV1alpha2().TLSRoutes(metav1.NamespaceAll).List(ctx, metav1.ListOptions{}); err == nil {
				for i := range tr.Items {
					links = append(links, linksForTLSRoute(&tr.Items[i])...)
				}
			} else if !isGatewayAPIMissing(err) {
				data.Error = err.Error()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = tpl.Execute(w, data)
				return
			}
		}

		sort.Slice(links, func(i, j int) bool {
			return strings.ToLower(links[i].Display) < strings.ToLower(links[j].Display)
		})
		data.Links = links
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tpl.Execute(w, data)
	})
}

func isGatewayAPIMissing(err error) bool {
	if err == nil {
		return false
	}
	if meta.IsNoMatchError(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "could not find the requested resource") ||
		strings.Contains(msg, "the server doesn't have a resource type")
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
	case '.', '-':
		return true
	default:
		return false
	}
}

func linkAnnotationValues(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}
	out := make(map[string]string)
	for k, v := range annotations {
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

func sslPassthroughFromAnnotations(annotations map[string]string) bool {
	if annotations == nil {
		return false
	}
	for _, key := range []string{
		"nginx.ingress.kubernetes.io/ssl-passthrough",
		"ingress.kubernetes.io/ssl-passthrough",
		"haproxy.org/ssl-passthrough",
	} {
		if strings.EqualFold(annotations[key], "true") {
			return true
		}
	}
	return false
}

func schemeForHost(ing *networkingv1.Ingress, host string) string {
	if hostInTLSSpec(ing, host) || sslPassthroughFromAnnotations(ing.Annotations) {
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

func buildURLFromScheme(scheme, host, path string) string {
	return scheme + "://" + joinHostPath(host, path)
}

func linksForIngress(ing *networkingv1.Ingress) []IngressLink {
	host := firstIngressHost(ing)
	ann := linkAnnotationValues(ing.Annotations)

	if len(ann) > 0 {
		return linksFromAnnotationMap(host, ann, schemeForHost(ing, host))
	}

	if host == "" {
		return nil
	}
	scheme := schemeForHost(ing, host)
	return []IngressLink{{
		Display: displayFromHost(host),
		Target:  buildURLFromScheme(scheme, host, ""),
	}}
}
