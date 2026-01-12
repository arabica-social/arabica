# Lexicon Schemas

## Record Types

Arabica defines 5 lexicon schemas:

### social.arabica.alpha.bean
Coffee bean records with origin, roast level, process, and roaster reference.

### social.arabica.alpha.roaster  
Coffee roaster records with name, location, and website.

### social.arabica.alpha.grinder
Grinder records with type (hand/electric), burr type (conical/flat), and notes.

### social.arabica.alpha.brewer
Brewing device records with name and description.

### social.arabica.alpha.brew
Brew session records including:
- Bean reference (AT-URI)
- Brewing parameters (temperature, time, water, coffee amounts)
- Grinder and brewer references (optional)
- Grind size, method, tasting notes, rating
- Pours array (embedded, not separate records)

## Design Decisions

### References
All references use AT-URIs pointing to user's own records.
Example: `at://did:plc:abc123/social.arabica.alpha.bean/3jxy123`

### Temperature Storage
Stored as integer in tenths of degrees Celsius.
Example: 935 represents 93.5°C

### Pours
Embedded in brew records as an array rather than separate collection.

### Required Fields
Minimal requirements - most fields are optional for flexibility.

## Schema Files

See `lexicons/` directory for complete JSON schemas.
