// Explicit paid access for pre-existing browser workflows. Membership-specific
// tests supply their own trial/expiry policy instead of using this fixture.
export const paidMembership = Object.freeze({
  tier: "premium",
  state: "paid",
  operational: true,
  premium: true,
  locations: 3,
  terminals_per_location: 5,
  valid_until: "2030-01-01T00:00:00Z",
});
