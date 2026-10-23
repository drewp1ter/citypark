// Set the theme before first paint to avoid a flash. Kept as a file (not inline) so the CSP can use script-src 'self'.
let theme
try {
	theme = localStorage.getItem('citypark-admin-theme')
} catch {
	// Storage can be blocked; fall back to the system theme.
	theme = null
}
if (theme !== 'light' && theme !== 'dark') {
	theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}
document.documentElement.classList.toggle('dark', theme === 'dark')
document.documentElement.setAttribute('data-theme', theme)
