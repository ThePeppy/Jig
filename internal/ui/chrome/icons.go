package chrome

import "github.com/egoist/mygo/ui"

var (
	// IconBrand is the Jig geometric prism emblem.
	IconBrand = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M10 2L17 6V14L10 18L3 14V6L10 2Z" stroke="#FFFFFF" stroke-width="1.7" stroke-linejoin="round"/>
		<path d="M10 2V10M10 10L17 6M10 10L3 6M10 10V18" stroke="#FFFFFF" stroke-width="1.4" stroke-linejoin="round" stroke-opacity="0.75"/>
	</svg>`))

	// IconLibrary represents the binary & database catalog.
	IconLibrary = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M10 3L17 6.5L10 10L3 6.5L10 3Z" stroke="#FFFFFF" stroke-width="1.65" stroke-linejoin="round"/>
		<path d="M3 10L10 13.5L17 10" stroke="#FFFFFF" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M3 13.5L10 17L17 13.5" stroke="#FFFFFF" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`))

	// IconTrace represents live MCP call telemetry & audit stream.
	IconTrace = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M2.5 10H5.5L7.8 4.5L12.2 15.5L14.5 10H17.5" stroke="#FFFFFF" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`))

	// IconSettings represents control center configuration.
	IconSettings = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M3 6H11M15 6H17M3 14H5M9 14H17" stroke="#FFFFFF" stroke-width="1.65" stroke-linecap="round"/>
		<circle cx="13" cy="6" r="2.2" stroke="#FFFFFF" stroke-width="1.65"/>
		<circle cx="7" cy="14" r="2.2" stroke="#FFFFFF" stroke-width="1.65"/>
	</svg>`))

	// IconPlus is the crisp add icon.
	IconPlus = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M10 4.2V15.8M4.2 10H15.8" stroke="#FFFFFF" stroke-width="1.8" stroke-linecap="round"/>
	</svg>`))

	// IconPlay is the launch analysis icon.
	IconPlay = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M6.5 4.5L15.5 10L6.5 15.5V4.5Z" fill="#FFFFFF" stroke="#FFFFFF" stroke-width="1.4" stroke-linejoin="round"/>
	</svg>`))

	// IconBinary represents an executable, firmware, or raw binary image.
	IconBinary = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M5 3.5H12L15.5 7V16.5C15.5 17.05 15.05 17.5 14.5 17.5H5C4.45 17.5 4 17.05 4 16.5V4.5C4 3.95 4.45 3.5 5 3.5Z" stroke="#FFFFFF" stroke-width="1.6" stroke-linejoin="round"/>
		<path d="M7.2 10.5L5.8 12L7.2 13.5M12.3 10.5L13.7 12L12.3 13.5M10.4 9.8L9.1 14.2" stroke="#FFFFFF" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`))

	// IconDatabase represents an IDA database (.i64 / .idb).
	IconDatabase = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<ellipse cx="10" cy="5.2" rx="6" ry="2.4" stroke="#FFFFFF" stroke-width="1.6"/>
		<path d="M4 5.2V10C4 11.33 6.69 12.4 10 12.4C13.31 12.4 16 11.33 16 10V5.2" stroke="#FFFFFF" stroke-width="1.6"/>
		<path d="M4 10V14.8C4 16.13 6.69 17.2 10 17.2C13.31 17.2 16 16.13 16 14.8V10" stroke="#FFFFFF" stroke-width="1.6"/>
	</svg>`))

	// IconAlert represents a missing file or error state.
	IconAlert = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M10 3.2L17.2 16H2.8L10 3.2Z" stroke="#FFFFFF" stroke-width="1.65" stroke-linejoin="round"/>
		<path d="M10 8V11.5M10 13.8V14" stroke="#FFFFFF" stroke-width="1.75" stroke-linecap="round"/>
	</svg>`))

	// IconTrash represents removing an item or clearing logs.
	IconTrash = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M4 6H16M7.5 6V4.2C7.5 3.8 7.8 3.5 8.2 3.5H11.8C12.2 3.5 12.5 3.8 12.5 4.2V6M5.5 6L6.2 15.8C6.25 16.4 6.75 16.8 7.35 16.8H12.65C13.25 16.8 13.75 16.4 13.8 15.8L14.5 6" stroke="#FFFFFF" stroke-width="1.55" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`))

	// IconFolder represents picking a directory.
	IconFolder = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M3 5.5C3 4.67 3.67 4 4.5 4H8.2L10 6H15.5C16.33 6 17 6.67 17 7.5V14.5C17 15.33 16.33 16 15.5 16H4.5C3.67 16 3 15.33 3 14.5V5.5Z" stroke="#FFFFFF" stroke-width="1.6" stroke-linejoin="round"/>
	</svg>`))

	// IconRefresh represents re-probing the local runtime.
	IconRefresh = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<path d="M16 10C16 13.31 13.31 16 10 16C6.69 16 4 13.31 4 10C4 6.69 6.69 4 10 4C12.2 4 14.1 5.18 15.15 6.95M15.5 4V7.2H12.3" stroke="#FFFFFF" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round"/>
	</svg>`))

	// IconCopy represents copying configuration to clipboard.
	IconCopy = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<rect x="7" y="7" width="9.5" height="9.5" rx="2" stroke="#FFFFFF" stroke-width="1.6"/>
		<path d="M13 7V5C13 4.17 12.33 3.5 11.5 3.5H5C4.17 3.5 3.5 4.17 3.5 5V11.5C3.5 12.33 4.17 13 5 13H7" stroke="#FFFFFF" stroke-width="1.6" stroke-linecap="round"/>
	</svg>`))

	// IconAgent represents external AI Agent MCP connectivity.
	IconAgent = ui.MustParseSVG([]byte(`<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
		<circle cx="6" cy="6" r="2.2" stroke="#FFFFFF" stroke-width="1.6"/>
		<circle cx="14" cy="6" r="2.2" stroke="#FFFFFF" stroke-width="1.6"/>
		<circle cx="10" cy="14.5" r="2.2" stroke="#FFFFFF" stroke-width="1.6"/>
		<path d="M7.8 7.3L9.1 12.5M12.2 7.3L10.9 12.5M8.2 6H11.8" stroke="#FFFFFF" stroke-width="1.5" stroke-linecap="round"/>
	</svg>`))
)
