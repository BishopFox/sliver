package pagerduty

import "context"

// ListAbilityResponse is the response when calling the ListAbility API endpoint.
type ListAbilityResponse struct {
	Abilities []string `json:"abilities"`
}

// ListAbilities lists all abilities on your account.
//
// Deprecated: Use ListAbilitiesWithContext instead.
func (c *Client) ListAbilities() (*ListAbilityResponse, error) {
	return c.ListAbilitiesWithContext(context.Background())
}

// ListAbilitiesWithContext lists all abilities on your account.
func (c *Client) ListAbilitiesWithContext(ctx context.Context) (*ListAbilityResponse, error) {
	resp, err := c.get(ctx, "/abilities", nil)
	if err != nil {
		return nil, err
	}

	var result ListAbilityResponse
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TestAbility checks if your account has the given ability.
//
// Deprecated: Use TestAbilityWithContext instead.
func (c *Client) TestAbility(ability string) error {
	return c.TestAbilityWithContext(context.Background(), ability)
}

// TestAbilityWithContext checks if your account has the given ability.
func (c *Client) TestAbilityWithContext(ctx context.Context, ability string) error {
	_, err := c.get(ctx, "/abilities/"+ability, nil)
	return err
}
