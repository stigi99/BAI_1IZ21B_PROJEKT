package views

// cmdInjectionCodeDiff renders the example vulnerable-versus-secure command
// execution snippet used by the command-injection lab page.
func cmdInjectionCodeDiff() string {
	return `// Vulnerable
cmd := exec.Command("sh", "-c", "ping -c1 "+host)

// Secure
cmd := exec.Command("ping", "-c1", host)
if !hostRegex.MatchString(host) {
    return error
}`
}

// cmdInjectionCurlExamples renders example curl commands for the public demo.
func cmdInjectionCurlExamples() string {
	return `# Vulnerable endpoint
curl "http://localhost:8080/api/ping-vulnerable?host=8.8.8.8"
curl "http://localhost:8080/api/ping-vulnerable?host=8.8.8.8%20;%20cat%20/etc/passwd"

# Secure endpoint
curl "http://localhost:8080/api/ping-secure?host=8.8.8.8"
curl "http://localhost:8080/api/ping-secure?host=8.8.8.8%20;%20cat%20/etc/passwd"`
}
