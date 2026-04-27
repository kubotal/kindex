In this web server application, add a new handler in serve.go which will fetch 
all ingresses resources of the connected kubernetes cluster and display them in a single page.

The page will display a link entry per ingress.

If there is no specific annotation (see below), the displayed link will be the first part of the host name (The part before the first dot) and the target the ingress host.

The application will lookup inside the ingress definition te set the scheme of the target. (NB: take care also of TLS passthroughs) 

The ingress may also include one or more annotation beginning with 'kindex.kubotal.io/link'.

The content of this annotation is a string made of one, two or three part, delimited by ':'.

- The first part will be the display link.
- The second part (optional) will be a path added the the ingress host to build the target.
- The third part (optional) will be a short description, to be displayed after the link.

There may be several such annotation. In this case, there will be several links for the ingress

If there is a single annotation with an empty string or an empty first part, then this ingress will not be displayed

Use go html templating for page layout.

The links will open in a new tab.

Link displayed in alphabetical order of the display link.

Keep things as simple as possible. 

------

I have added a --kubeconfig option and set the serve command to NoArg.

Modify the code to find k8s configuration the following way:
- If --kubeconfig option is set, use it.
- if not, if there is a KUBECONFIG environment variable KUBECONFIG, use it
- if not lookup in the user configuration (~/.kube/config)

------

Got kubeconfig: invalid configuration: no configuration has been provided, try setting KUBERNETES_MASTER environment variable
Seems ~/.kube/config is not taken in account


----
Which ingress use the "ingress.kubernetes.io/ssl-passthrough" annotation ?

-----
I was wondering if you know an existing controller which use the annotation
---
The link you provided about HAProxy give Page Not Found
From HAProxy doc (https://www.haproxy.com/documentation/kubernetes-ingress/community/configuration-reference/ingress/#ssl-passthrough), the annotation is haproxy.org/ssl-passthrough: "true"

-----

    You're right, sorry about the broken link. And thanks for the correction on the official HAProxy controller — it uses haproxy.org/ssl-passthrough, not ingress.kubernetes.io/ssl-passthrough.
    
    To be precise, there are actually two distinct HAProxy-based ingress controllers:
    
    HAProxy Kubernetes Ingress Controller (official, by HAProxy Technologies, at haproxy.com) — uses haproxy.org/ prefix. So: haproxy.org/ssl-passthrough: "true".
    
    HAProxy Ingress (community, by jcmoraisjr/haproxy-ingress) — uses haproxy-ingress.github.io/ by default, and documents ingress.kubernetes.io/ as a backward-compatible prefix. This is the one I was referring to earlier.
    
    So to answer honestly: ingress.kubernetes.io/ssl-passthrough is only supported by the community HAProxy Ingress project for backward compatibility — it's not a widely used key. No mainstream, actively maintained controller uses it as a primary or default annotation.
    
    Bottom line: the ingress.kubernetes.io/ssl-passthrough check in our code is of marginal value. Want me to remove it and keep only nginx.ingress.kubernetes.io/ssl-passthrough (which is the one people actually use)?

----
No. I want to handle all cases. So.keep them and add the  haproxy.org/ssl-passthrough: "true"


-----

Now could you get the cluster name and display in the title and the first line of the page.


----
I change my mind: Don't display the link only if the annotation value is "". if it is like ":pathValue" or "::comment", display it 

-----

The '/' is not allowed by k8s in the annotation key. Remove it from the 3 separators

-----
Display global.version in the web page.in an hidden way (Only visible by view source)

No.  It does not appears in view page source

-----
I have added a '--mode' option. Ensure it is either 'dark' or 'light' and adjust page look accordingly 

----

Complete the helm chart by an appropriate rbac.yaml (serviceAccount, role/clusterRole and roleBinding)

----

Redact a user oriented README.md

---

Now, we want also to support Gateway API based ingresses. So, lookup in HTTPRoute and TLSRoute resources and add related entries to the list.
Support the same annotation.

You miss the case where HTTPRoute is using https. I think you should retrieve anc check the associated gateway listener to grab the info.