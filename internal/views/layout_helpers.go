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

func authHeroTip(mode string) string {
	if mode == "register" {
		return "Tip: open the ⚔️ PAYLOADS drawer on the right edge — every payload is one click away."
	}
	return "Tip: in vulnerable mode the login accepts any password for an existing user (Broken Auth demo)."
}
