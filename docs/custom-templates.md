# Custom Templates

To customize the HTML pages rendered by Gangly, you may provide a set of custom templates to use instead of the built-in ones.

:exclamation: **Important: The data passed to the templates might change between versions, and we do not guarantee that we will maintain backwards compatibility. If using custom templates, extra care must be taken when upgrading Gangly.**

To enable this feature, set the `customHTMLTemplatesDir` option in Gangly's configuration file to a directory that contains the following custom templates:

* home.tmpl: Home page template.
* commandline.tmpl: Post-login template that typically lists the commands needed to configure `kubectl`.

The templates are processed using Go's `html/template` [package][0].

Assets to be used by custom templates can be pointed to by setting `customAssetsDir`. The contents will be served
under /assets/

[0]: https://golang.org/pkg/html/template/
