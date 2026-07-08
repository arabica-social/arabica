{
  "id": "ba3ec43a",
  "title": "P2.3: Port brew view page",
  "tags": [],
  "status": "completed",
  "created_at": "2026-07-08T17:07:11.495Z"
}

Port the brew view to web/src/routes/brews/[actor]/[id]/+page.svelte. Custom layout (no EntityViewLayout): RecordViewHeader, brew summary (rating + ratio/time/method stats), bean reference card, inputs journal (coffee/water/grinder/grind size/temp/filter), process section (brewer/time/espresso/pourover params), recipe section, pours, tasting notes, save-as-recipe button for owner, ActionBar, CommentSection. Consume GET /api/brews/{actor}/{id}.
