package e2e

import (
	"testing"
)

func TestOnboardingFlow(t *testing.T) {
	page := browser.MustPage(testServer.URL + "/")
	defer page.MustClose()

	// Wait until the initial command center UI loads (the input elements exist)
	page.MustWaitElementsMoreThan("input[name='role']", 0)

	// The mock database is empty, so it renders onboarding.html
	// Wait for the form
	page.MustWaitElementsMoreThan("input[name='role']", 0)

	// Fill out the onboarding form
	page.MustElement("input[name='role']").MustInput("Senior Engineer")
	page.MustElement("input[name='industry']").MustInput("Tech")
	page.MustElement("select[name='experience_level']").MustSelect("Expert")

	// Submit form
	page.MustElement("button[type='submit']").MustClick()

	// Wait for the Command Center to replace the modal
	// Since handleOnboard renders command_center.html which contains textarea[name='prompt']
	page.MustWaitElementsMoreThan("textarea[name='prompt']", 0)

	// Verify we are now on the command center
	content := page.MustElement("body").MustText()
	if len(content) == 0 {
		t.Errorf("Expected page content, got empty string")
	}
}
