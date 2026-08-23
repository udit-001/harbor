// node --test suite for HarborCore (the pure comment-surface core).
// Run via the Go wrapper in theme_test.go… actually pageview_core_test.go:
// `go test ./internal/render/` shells out here when node is available.
const test = require('node:test');
const assert = require('node:assert');
const Core = require('./pageview-core.js');

test('exIdOf parses excalidraw anchor paths and rejects others', () => {
  assert.equal(Core.exIdOf('excalidraw:box-a'), 'box-a');
  assert.equal(Core.exIdOf('excalidraw:'), '');
  assert.equal(Core.exIdOf('.row > p'), null);
  assert.equal(Core.exIdOf(null), null);
  assert.equal(Core.exIdOf(''), null);
  // prefix must match exactly — a CSS path that merely contains it is not one
  assert.equal(Core.exIdOf('div[data-x="excalidraw:box"]'), null);
});

test('anchorKindFor maps compose types to stored kinds', () => {
  assert.equal(Core.anchorKindFor('selection'), 'text');
  assert.equal(Core.anchorKindFor('element'), 'element');
  assert.equal(Core.anchorKindFor('general'), 'document');
  assert.equal(Core.anchorKindFor(undefined), 'document');
});

test('stateToAnchor shapes compose state for storage', () => {
  assert.deepEqual(
    Core.stateToAnchor({ type: 'element', anchor: 'excalidraw:r1', quote: '' }),
    { kind: 'element', path: 'excalidraw:r1', quote: '' }
  );
  assert.deepEqual(
    Core.stateToAnchor({ type: 'selection', anchor: '#p1', quote: 'hello' }),
    { kind: 'text', path: '#p1', quote: 'hello' }
  );
});

test('firstAnchorLabel reads like a human wrote it', () => {
  assert.equal(Core.firstAnchorLabel({ kind: 'text', quote: 'fix this' }), 'fix this');
  assert.equal(Core.firstAnchorLabel({ kind: 'text', quote: '', path: '#p' }), '#p');
  assert.equal(Core.firstAnchorLabel({ kind: 'text' }), 'text selection');
  assert.equal(Core.firstAnchorLabel({ kind: 'element', path: 'excalidraw:r1' }), 'shape in drawing');
  assert.equal(Core.firstAnchorLabel({ kind: 'element', path: '#id' }), '#id');
  assert.equal(Core.firstAnchorLabel({ kind: 'element', path: '' }), 'element');
  assert.equal(Core.firstAnchorLabel({ kind: 'document' }), 'whole page');
  assert.equal(Core.firstAnchorLabel(null), '');
});

test('anchorLabel summarizes single, multi, and legacy comments', () => {
  assert.equal(Core.anchorLabel({ anchors: [{ kind: 'document' }] }), 'whole page');
  assert.equal(Core.anchorLabel({ anchors: [{}, {}] }), '2 spots');
  assert.equal(Core.anchorLabel({ quote: 'legacy quote' }), 'legacy quote');
  assert.equal(Core.anchorLabel({ type: 'selection', anchor: '#p' }), '#p');
  assert.equal(Core.anchorLabel({}), 'whole page');
});

test('mode coordinator is exclusive and announces changes', () => {
  const seen = [];
  const m = Core.createModes(d => seen.push(d));
  assert.equal(m.get(), 'reader');
  m.set('comment');
  assert.equal(m.get(), 'comment');
  m.set('comment'); // no-op: same mode
  assert.equal(seen.length, 1);
  assert.deepEqual(seen[0], { prev: 'reader', next: 'comment' });
  m.set('reader');
  assert.deepEqual(seen[1], { prev: 'comment', next: 'reader' });
});

test('mode coordinator works without a dispatcher', () => {
  const m = Core.createModes();
  m.set('tour');
  assert.equal(m.get(), 'tour');
});
