package views

func layoutBodyClass(securityEnabled bool) string {
	const base = "min-h-screen flex flex-col bg-shell bg-fixed text-slate-900 antialiased "
	if securityEnabled {
		return base + "sec-secure"
	}
	return base + "sec-vuln"
}

func commentPlaceholder(securityEnabled bool) string {
	if securityEnabled {
		return "Comment text — HTML will be escaped"
	}
	return "Try <script>alert('XSS')</script>"
}
