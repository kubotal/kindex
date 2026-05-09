package handlers

import (
	"sort"
	"strings"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func parentRefGroupString(ref gatewayv1.ParentReference) string {
	if ref.Group == nil {
		return gatewayv1.GroupName
	}
	return string(*ref.Group)
}

func parentRefKindString(ref gatewayv1.ParentReference) string {
	if ref.Kind == nil {
		return "Gateway"
	}
	return string(*ref.Kind)
}

func parentRefIsGateway(ref gatewayv1.ParentReference) bool {
	if parentRefKindString(ref) != "Gateway" {
		return false
	}
	return parentRefGroupString(ref) == gatewayv1.GroupName
}

func gatewayKeyForParent(routeNamespace string, ref gatewayv1.ParentReference) string {
	ns := routeNamespace
	if ref.Namespace != nil {
		ns = string(*ref.Namespace)
	}
	return ns + "/" + string(ref.Name)
}

func wildcardCovers(pattern, host string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return strings.EqualFold(pattern, host)
	}
	suffix := pattern[2:]
	if strings.EqualFold(host, suffix) {
		return false
	}
	return strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(suffix))
}

func hostnamePatternsIntersect(a, b string) bool {
	if a == "" || b == "" {
		return true
	}
	aWild := strings.HasPrefix(a, "*.")
	bWild := strings.HasPrefix(b, "*.")
	if aWild && bWild {
		return true
	}
	if aWild {
		return wildcardCovers(a, b)
	}
	if bWild {
		return wildcardCovers(b, a)
	}
	return strings.EqualFold(a, b)
}

func listenerHostnameRelevantForRoute(listenerHost *gatewayv1.Hostname, routeHosts []gatewayv1.Hostname) bool {
	if listenerHost == nil || string(*listenerHost) == "" {
		return true
	}
	if len(routeHosts) == 0 {
		return true
	}
	lh := string(*listenerHost)
	for _, rh := range routeHosts {
		if hostnamePatternsIntersect(lh, string(rh)) {
			return true
		}
	}
	return false
}

func listenerMatchesParentRef(l gatewayv1.Listener, ref gatewayv1.ParentReference) bool {
	if ref.SectionName != nil && l.Name != *ref.SectionName {
		return false
	}
	if ref.Port != nil && l.Port != *ref.Port {
		return false
	}
	return true
}

func listenerProtocolRelevantForHTTPRoute(l gatewayv1.Listener, ref gatewayv1.ParentReference) bool {
	// With an explicit sectionName the route targets that listener; honor its protocol (including TLS).
	if ref.SectionName != nil {
		switch l.Protocol {
		case gatewayv1.HTTPProtocolType, gatewayv1.HTTPSProtocolType, gatewayv1.TLSProtocolType:
			return true
		default:
			return false
		}
	}
	// Without sectionName, avoid treating unrelated TLS-only listeners as applying to this HTTPRoute.
	switch l.Protocol {
	case gatewayv1.HTTPProtocolType, gatewayv1.HTTPSProtocolType:
		return true
	default:
		return false
	}
}

func listenersForHTTPRouteOnGateway(gw *gatewayv1.Gateway, ref gatewayv1.ParentReference, routeHostnames []gatewayv1.Hostname) []gatewayv1.Listener {
	var out []gatewayv1.Listener
	for _, l := range gw.Spec.Listeners {
		if !listenerMatchesParentRef(l, ref) {
			continue
		}
		if !listenerProtocolRelevantForHTTPRoute(l, ref) {
			continue
		}
		if !listenerHostnameRelevantForRoute(l.Hostname, routeHostnames) {
			continue
		}
		out = append(out, l)
	}
	return out
}

func httpsFromGatewayListeners(route *gatewayv1.HTTPRoute, gateways map[string]*gatewayv1.Gateway) bool {
	if len(gateways) == 0 || len(route.Spec.ParentRefs) == 0 {
		return false
	}
	hosts := route.Spec.Hostnames
	for _, ref := range route.Spec.ParentRefs {
		if !parentRefIsGateway(ref) {
			continue
		}
		key := gatewayKeyForParent(route.Namespace, ref)
		gw := gateways[key]
		if gw == nil {
			continue
		}
		for _, l := range listenersForHTTPRouteOnGateway(gw, ref, hosts) {
			if l.Protocol == gatewayv1.HTTPSProtocolType || l.Protocol == gatewayv1.TLSProtocolType {
				return true
			}
		}
	}
	return false
}

func firstHTTPRouteHostname(hostnames []gatewayv1.Hostname) string {
	for _, h := range hostnames {
		s := string(h)
		if s == "" || strings.HasPrefix(s, "*.") {
			continue
		}
		return s
	}
	return ""
}

func linksForHTTPRoute(route *gatewayv1.HTTPRoute, gateways map[string]*gatewayv1.Gateway) []IngressLink {
	host := firstHTTPRouteHostname(route.Spec.Hostnames)
	ann := linkAnnotationValues(route.Annotations)
	scheme := schemeHTTPRouteString(route, gateways)
	if len(ann) > 0 {
		return linksFromAnnotationMap(host, ann, scheme)
	}
	if host == "" {
		return nil
	}
	return []IngressLink{{
		Display: displayFromHost(host),
		Target:  buildURLFromScheme(scheme, host, ""),
	}}
}

func linksForTLSRouteV1(route *gatewayv1.TLSRoute) []IngressLink {
	host := firstHTTPRouteHostname(route.Spec.Hostnames)
	ann := linkAnnotationValues(route.Annotations)
	scheme := schemeTLSRouteString()
	if len(ann) > 0 {
		return linksFromAnnotationMap(host, ann, scheme)
	}
	if host == "" {
		return nil
	}
	return []IngressLink{{
		Display: displayFromHost(host),
		Target:  buildURLFromScheme(scheme, host, ""),
	}}
}

func firstTLSRouteAlpha2Hostname(hostnames []gatewayv1alpha2.Hostname) string {
	for _, h := range hostnames {
		s := string(h)
		if s == "" || strings.HasPrefix(s, "*.") {
			continue
		}
		return s
	}
	return ""
}

// linksForTLSRouteAlpha2 handles gateway.networking.k8s.io/v1alpha2 TLSRoute (older clusters).
func linksForTLSRouteAlpha2(route *gatewayv1alpha2.TLSRoute) []IngressLink {
	host := firstTLSRouteAlpha2Hostname(route.Spec.Hostnames)
	ann := linkAnnotationValues(route.Annotations)
	scheme := schemeTLSRouteString()
	if len(ann) > 0 {
		return linksFromAnnotationMap(host, ann, scheme)
	}
	if host == "" {
		return nil
	}
	return []IngressLink{{
		Display: displayFromHost(host),
		Target:  buildURLFromScheme(scheme, host, ""),
	}}
}

func linksFromAnnotationMap(host string, ann map[string]string, scheme string) []IngressLink {
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
		if display == "" {
			display = displayFromHost(host)
		}
		out = append(out, IngressLink{
			Display:     display,
			Target:      buildURLFromScheme(scheme, host, path),
			Description: desc,
		})
	}
	return out
}

func schemeHTTPRouteString(route *gatewayv1.HTTPRoute, gateways map[string]*gatewayv1.Gateway) string {
	if httpsFromGatewayListeners(route, gateways) {
		return "https"
	}
	if sslPassthroughFromAnnotations(route.Annotations) {
		return "https"
	}
	return "http"
}

func schemeTLSRouteString() string {
	return "https"
}
