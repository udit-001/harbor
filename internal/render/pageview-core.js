// HarborCore — the pure decision core of the pageview comment surface.
// Everything here is DOM-free so it can run under node --test; browser glue
// (event dispatch, iframe access) stays in pageview-annotations.js /
// pageview-tour.js and calls into this object. Loaded before those files.
(function (global) {
  'use strict';

  // ── Anchor vocabulary ──────────────────────────────────────────────────
  // An anchor is {kind, path, quote?}. kind: 'text' | 'element' | 'document'.
  // Excalidraw element anchors carry path "excalidraw:<elementId>".

  var EX_PREFIX = 'excalidraw:';

  // exIdOf returns the scene element id for an excalidraw anchor path,
  // or null for any other path.
  function exIdOf(path) {
    if (path && path.slice(0, EX_PREFIX.length) === EX_PREFIX) {
      return path.slice(EX_PREFIX.length);
    }
    return null;
  }

  // anchorKindFor maps a compose-state type to its stored anchor kind.
  function anchorKindFor(type) {
    return type === 'selection' ? 'text' : (type === 'element' ? 'element' : 'document');
  }

  // firstAnchorLabel is the human-readable "where" line for a comment's
  // first anchor.
  function firstAnchorLabel(a) {
    if (!a) return '';
    if (a.kind === 'text') return a.quote || a.path || 'text selection';
    if (a.kind === 'element') {
      var p = a.path || '';
      if (p.slice(0, EX_PREFIX.length) === EX_PREFIX) return 'shape in drawing';
      return p || 'element';
    }
    return 'whole page';
  }

  // anchorLabel summarizes a comment's anchoring for the list rows.
  function anchorLabel(c) {
    var a = (c && c.anchors) || [];
    if (a.length > 1) return a.length + ' spots';
    if (a.length === 1) return firstAnchorLabel(a[0]);
    if (c && c.quote) return c.quote;
    if (c && c.type === 'selection') return c.anchor || 'text selection';
    if (c && c.type === 'element') return c.anchor || 'element';
    return 'whole page';
  }

  // stateToAnchor maps compose state {type, anchor, quote} to the stored
  // anchor shape.
  function stateToAnchor(state) {
    return {
      kind: anchorKindFor(state && state.type),
      path: (state && state.anchor) || '',
      quote: (state && state.quote) || ''
    };
  }

  // ── Mode coordinator ───────────────────────────────────────────────────
  // READER | TOUR | COMMENT are mutually exclusive (HARB-31): one surface
  // owns the stage. The coordinator holds the current mode and announces
  // every change through dispatch({prev, next}) — the browser glue turns
  // that into a 'harbor-mode' event; tests assert transitions directly.

  function createModes(dispatch) {
    return {
      m: 'reader',
      get: function () { return this.m; },
      set: function (next) {
        if (next === this.m) return;
        var prev = this.m;
        this.m = next;
        if (dispatch) dispatch({ prev: prev, next: next });
      }
    };
  }

  var api = {
    EX_PREFIX: EX_PREFIX,
    exIdOf: exIdOf,
    anchorKindFor: anchorKindFor,
    firstAnchorLabel: firstAnchorLabel,
    anchorLabel: anchorLabel,
    stateToAnchor: stateToAnchor,
    createModes: createModes
  };

  global.HarborCore = api;
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
})(typeof window !== 'undefined' ? window : globalThis);
