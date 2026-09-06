# Detail Asset Content Frame

`DetailAssetContentFrame` owns the shared responsive content gutters and bottom
breathing room for the website, subdomain, IP address, and URL workspaces under
target detail and scan history detail routes.

Use it in the resolved route page and in the state-less route fallback. Do not
add it inside a domain `*LoadingState`; those loading states already render
inside the page-owned frame and must keep their domain-specific DataTable
geometry.

The default frame is `px-4 pb-4 md:pb-6 lg:px-6`. Route-specific attributes and
classes may be forwarded when they do not create another gutter owner.
