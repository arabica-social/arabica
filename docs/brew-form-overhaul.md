# Brew Form Overhaul

Reducing friction for new and returning users logging brews.

## 1. Kill the Mode Chooser

Remove the Recipe vs Freeform gate. Start directly in the form. Recipe
selection becomes an optional field at the top (collapsed or a small link).

## 2. Progressive Disclosure (Collapsible Sections)

- Only **Coffee** section is open by default
- **Brewing** and **Results** collapse into accordions with one-line summaries
  ("250g water, 93°C, 3:00" or "No details yet")
- Bean + Rating is the minimum viable brew — make that feel intentional, not
  like skipping the form

## 3. Inline Entity Creation (Typeahead Selects)

**Highest-impact change.** Replace the current select + "+ New" modal pattern
with a combo input that supports both searching existing entities and creating
new ones inline.

- User types in the bean/brewer/grinder field
- Matching entities appear as suggestions
- If no match, offer "Create [typed name]" as an option
- Selecting that creates a minimal record (just the name) on the fly
- User can flesh out details (origin, roast level, etc.) later from Manage page
- Eliminates the multi-step modal detour that blocks first-brew logging

## 4. Pre-seeded Brewer Suggestions

On first use (no brewers exist), show common brewers as quick-pick buttons:
V60, Chemex, Aeropress, French Press, Espresso Machine, Moka Pot.

One tap creates the entity with name + brewer type pre-filled. No modal, no
form fields.

## 5. Beverage Field

Add a `beverage` field to the brew record (separate from brew method):
Black, Latte, Cappuccino, Cortado, Americano, Flat White, Iced, etc.

This separates "how you extracted" from "what you made with it" — handles milk
drinks cleanly without polluting the brew method taxonomy.

## 6. Tasting Wheel (Future)

Structured tasting notes via scored axes (sweet, acidic, floral, body, etc.)
instead of/alongside free text. Enables comparison and visualization across
brews. Bigger lift — save for later.
