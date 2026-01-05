# Arabica Lexicon Schema Design

## Overview

This document describes the lexicon schemas for Arabica coffee tracking records in the AT Protocol ecosystem.

**Namespace:** `com.arabica.*`  
**Records:** Public by default (visible via repo exploration)  
**Key Format:** TID (Timestamp Identifier) - automatic

---

## Schema Decisions

### Reference Strategy
**Decision:** Use AT-URIs for all references between records

**Rationale:**
- Maintains decentralization (references work across any PDS)
- Standard atproto pattern
- Allows users to reference their own records
- Each user maintains their own copy of beans/roasters (duplicates OK)

**Example Reference:**
```
beanRef: "at://did:plc:abc123xyz/com.arabica.bean/3jxyabcd123"
```

### Pours Handling
**Decision:** Embed pours as an array within brew records

**Rationale:**
- Pours are tightly coupled to brews (no independent existence)
- Simpler than separate collection
- Reduces number of PDS queries
- Matches original SQLite schema pattern

**Alternative Considered:** Separate `com.arabica.pour` collection
- Rejected due to added complexity and query overhead

### Enum vs Free Text
**Decision:** Mix of both approaches

**Enums Used:**
- `grinder.grinderType`: `["hand", "electric", "electric_hand"]`
- `grinder.burrType`: `["conical", "flat", ""]` (empty string for unknown)

**Free Text Used:**
- `bean.roastLevel`: User may have custom roast descriptions
- `bean.process`: Processing methods vary
- `brew.method`: Brewing methods are diverse
- `brew.grindSize`: Can be numeric (grinder setting) or descriptive

**Rationale:**
- Enums for truly limited, well-defined sets
- Free text for user creativity and flexibility
- Matches current app behavior

### Field Mappings from SQLite

#### Removed Fields
- `user_id` - Implicit (records exist in user's DID repo)
- `id` - Replaced by TID (rkey)

#### Reference Changes
```
SQLite                  →  Lexicon
-----------------------------------------
bean.roaster_id (int)   →  bean.roasterRef (AT-URI)
brew.bean_id (int)      →  brew.beanRef (AT-URI)
brew.grinder_id (int)   →  brew.grinderRef (AT-URI)
brew.brewer_id (int)    →  brew.brewerRef (AT-URI)
```

#### Pours Changes
```
SQLite                     →  Lexicon
-------------------------------------------
pours table (separate)     →  brew.pours[] (embedded)
pour.brew_id (int)         →  (removed, part of brew)
pour.pour_number (int)     →  (removed, array index)
```

---

## Schema Definitions

### 1. `com.arabica.bean`
Coffee bean variety tracked by the user.

**Required Fields:**
- `name` (string, max 200 chars)
- `createdAt` (datetime)

**Optional Fields:**
- `origin` (string, max 200 chars) - Geographic origin
- `roastLevel` (string, max 100 chars) - Roast level description
- `process` (string, max 100 chars) - Processing method
- `description` (string, max 1000 chars) - Additional notes
- `roasterRef` (at-uri) - Reference to roaster record

**Example:**
```json
{
  "name": "Ethiopian Yirgacheffe",
  "origin": "Ethiopia",
  "roastLevel": "Light",
  "process": "Washed",
  "description": "Floral and citrus notes",
  "roasterRef": "at://did:plc:abc/com.arabica.roaster/3jxy...",
  "createdAt": "2024-01-04T12:00:00Z"
}
```

---

### 2. `com.arabica.roaster`
Coffee roaster company.

**Required Fields:**
- `name` (string, max 200 chars)
- `createdAt` (datetime)

**Optional Fields:**
- `location` (string, max 200 chars) - Location description
- `website` (uri, max 500 chars) - Roaster website URL

**Example:**
```json
{
  "name": "Blue Bottle Coffee",
  "location": "Oakland, CA",
  "website": "https://bluebottlecoffee.com",
  "createdAt": "2024-01-04T12:00:00Z"
}
```

---

### 3. `com.arabica.grinder`
Coffee grinder equipment.

**Required Fields:**
- `name` (string, max 200 chars)
- `createdAt` (datetime)

**Optional Fields:**
- `grinderType` (enum: `["hand", "electric", "electric_hand"]`)
- `burrType` (enum: `["conical", "flat", ""]`) - Empty string for unknown
- `notes` (string, max 1000 chars) - Additional notes

**Example:**
```json
{
  "name": "Baratza Encore",
  "grinderType": "electric",
  "burrType": "conical",
  "notes": "Great entry-level grinder",
  "createdAt": "2024-01-04T12:00:00Z"
}
```

---

### 4. `com.arabica.brewer`
Coffee brewing device or method.

**Required Fields:**
- `name` (string, max 200 chars)
- `createdAt` (datetime)

**Optional Fields:**
- `description` (string, max 1000 chars) - Description or notes

**Example:**
```json
{
  "name": "Hario V60",
  "description": "Size 02, ceramic",
  "createdAt": "2024-01-04T12:00:00Z"
}
```

---

### 5. `com.arabica.brew`
Coffee brewing session with parameters.

**Required Fields:**
- `beanRef` (at-uri) - Reference to bean used
- `createdAt` (datetime)

**Optional Fields:**
- `method` (string, max 100 chars) - Brewing method
- `temperature` (number, 0-100) - Water temperature in Celsius
- `waterAmount` (integer, ≥0) - Water amount in grams/ml
- `timeSeconds` (integer, ≥0) - Total brew time
- `grindSize` (string, max 50 chars) - Grind setting
- `grinderRef` (at-uri) - Reference to grinder
- `brewerRef` (at-uri) - Reference to brewer
- `tastingNotes` (string, max 2000 chars) - Tasting notes
- `rating` (integer, 1-10) - Rating
- `pours` (array of pour objects) - Pour schedule

**Pour Object:**
- `waterAmount` (integer, required) - Water in this pour
- `timeSeconds` (integer, required) - Time of pour

**Example:**
```json
{
  "beanRef": "at://did:plc:abc/com.arabica.bean/3jxy...",
  "method": "Pour Over",
  "temperature": 93,
  "waterAmount": 300,
  "timeSeconds": 180,
  "grindSize": "18",
  "grinderRef": "at://did:plc:abc/com.arabica.grinder/3jxy...",
  "brewerRef": "at://did:plc:abc/com.arabica.brewer/3jxy...",
  "tastingNotes": "Bright acidity, notes of lemon and bergamot",
  "rating": 9,
  "pours": [
    {"waterAmount": 50, "timeSeconds": 0},
    {"waterAmount": 100, "timeSeconds": 45},
    {"waterAmount": 150, "timeSeconds": 90}
  ],
  "createdAt": "2024-01-04T08:30:00Z"
}
```

---

## Data Relationships

```
Roaster ←── Bean ←── Brew
                      ↓
                   Grinder
                      ↓
                   Brewer
                      ↓
                   Pours (embedded)
```

**Key Points:**
- All relationships are AT-URI references
- References point within user's own repo (typically)
- Users maintain their own copies of beans/roasters
- Broken references should be handled gracefully in UI

---

## Validation Rules

### String Lengths
- Names/titles: 200 chars max
- Short fields: 50-100 chars max
- Descriptions/notes: 1000 chars max
- Tasting notes: 2000 chars max
- URLs: 500 chars max

### Numeric Constraints
- Temperature: 0-100°C (reasonable coffee range)
- Rating: 1-10 (explicit range)
- Water amount, time: ≥0 (non-negative)

### Required Fields
- All records: `createdAt` (for temporal ordering)
- Bean: `name` (minimum identifier)
- Roaster: `name`
- Grinder: `name`
- Brewer: `name`
- Brew: `beanRef` (essential relationship)

---

## Schema Evolution

### Backward Compatibility
If schema changes are needed:
- Add new optional fields freely
- Never remove required fields
- Deprecate fields gradually
- Use new collection IDs for breaking changes

### Versioning Strategy
- Start with `com.arabica.*` (implicit v1)
- If major breaking change needed: `com.arabica.v2.*`
- Document migration paths

---

## Publishing Lexicons

### Phase 1 (Current)
- Lexicons in project repo: `lexicons/`
- Not yet published publicly
- For development use

### Phase 2 (Future)
Host lexicons at one of:
1. **GitHub Raw** (easiest):
   ```
   https://raw.githubusercontent.com/user/arabica/main/lexicons/com.arabica.brew.json
   ```

2. **Domain .well-known** (proper):
   ```
   https://arabica.com/.well-known/atproto/lexicons/com.arabica.brew.json
   ```

3. **Both** (recommended):
   - GitHub for development/reference
   - Domain for production validation

---

## Testing Strategy

### Manual Testing (Phase 0)
- [ ] Create test records with atproto CLI
- [ ] Verify schema validation passes
- [ ] Test all field types
- [ ] Test reference resolution
- [ ] Test with missing optional fields

### Automated Testing (Future)
- Unit tests for record conversion
- Integration tests for CRUD operations
- Schema validation in CI/CD

---

## Open Questions

1. **Should we version lexicons explicitly?**
   - Current: No version in ID
   - Alternative: `com.arabica.v1.brew`

2. **Should we add metadata fields?**
   - Examples: `version`, `updatedAt`, `tags`
   - Current: Minimal schema

3. **Should we support recipe sharing?**
   - Could add `recipeRef` field to brew
   - Future enhancement

---

## Comparison to Original Schema

### SQLite Tables → Lexicons

| SQLite Table | Lexicon Collection | Changes |
|--------------|-------------------|---------|
| `users` | (removed) | Implicit - DID is the user |
| `beans` | `com.arabica.bean` | `roaster_id` → `roasterRef` |
| `roasters` | `com.arabica.roaster` | Minimal changes |
| `grinders` | `com.arabica.grinder` | Enum types added |
| `brewers` | `com.arabica.brewer` | Minimal changes |
| `brews` | `com.arabica.brew` | Foreign keys → AT-URIs |
| `pours` | (embedded in brew) | Now part of brew record |

### Key Architectural Changes
- No user_id (implicit from repo)
- No integer IDs (TID-based keys)
- No foreign key constraints (AT-URI references)
- No separate pours table (embedded)
- Public by default (repo visibility)

---

## Migration Notes

When migrating from SQLite to atproto:
1. User ID 1 → Your DID
2. Integer IDs → Create new TIDs
3. Foreign keys → Look up AT-URIs
4. Pours → Embed in brew records
5. Timestamps → RFC3339 format

See `docs/migration-guide.md` (future) for detailed migration script.
