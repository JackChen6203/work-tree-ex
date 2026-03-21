import { describe, expect, it } from "vitest";
import { aiDrafts, budgetCategories, itineraryDays, trips } from "./mock-data";

describe("frontend mock data contract", () => {
  it("provides at least one trip with route-ready identity fields", () => {
    expect(trips.length).toBeGreaterThan(0);
    expect(trips[0]).toMatchObject({
      id: expect.any(String),
      name: expect.any(String),
      destination: expect.any(String),
      timezone: expect.any(String)
    });
  });

  it("keeps itinerary items grouped by day with time and place metadata", () => {
    expect(itineraryDays.length).toBeGreaterThan(0);
    for (const day of itineraryDays) {
      expect(day.items.length).toBeGreaterThan(0);
      for (const item of day.items) {
        expect(item.time).toContain("-");
        expect(item.location.length).toBeGreaterThan(0);
      }
    }
  });

  it("keeps budget and AI planner fixtures usable for dashboard summaries", () => {
    const estimated = budgetCategories.reduce((sum, item) => sum + item.estimated, 0);
    expect(estimated).toBeGreaterThan(0);
    expect(aiDrafts.every((draft) => draft.score > 0 && draft.score <= 100)).toBe(true);
  });
});
