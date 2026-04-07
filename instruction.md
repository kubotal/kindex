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
