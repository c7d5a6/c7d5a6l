-- Normalize Liquipedia player links: keep MediaWiki-style literal parentheses.
-- Go's url.URL previously stored Larva_%28Player%29; lookups used Larva_(Player).
UPDATE player
SET link = REPLACE(REPLACE(link, '%28', '('), '%29', ')')
WHERE link LIKE '%\%28%' ESCAPE '\';
