/// Flutter Web Turnstile widget. Mirrors React `turnstile-widget.tsx`: the
/// official script is loaded once per page, the challenge renders into a
/// platform-view `<div>`, and the widget is removed on dispose.
library;

import 'dart:async';
import 'dart:js_interop';
import 'dart:js_interop_unsafe';

import 'package:flutter/widgets.dart';
import 'package:web/web.dart' as web;

const _scriptSrc =
    'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';

Future<void>? _scriptLoad;

/// Whether the built-in Turnstile widget can render here.
bool get turnstileSupported => true;

Future<void> _loadScript() {
  if (globalContext.has('turnstile')) return Future.value();
  return _scriptLoad ??= () {
    final completer = Completer<void>();
    final script = web.HTMLScriptElement()
      ..src = _scriptSrc
      ..async = true
      ..defer = true;
    script.onload = ((web.Event _) => completer.complete()).toJS;
    script.onerror = ((web.Event _) {
      // Let a later mount retry rather than caching the failure.
      _scriptLoad = null;
      completer.completeError(StateError('Failed to load Turnstile'));
    }).toJS;
    web.document.head!.appendChild(script);
    return completer.future;
  }();
}

/// Renders a Turnstile challenge and reports its token.
class TurnstileView extends StatefulWidget {
  /// Cloudflare Turnstile site key.
  final String siteKey;

  /// Called with a token whenever the challenge completes.
  final ValueChanged<String> onToken;

  /// Called when an issued token expires.
  final VoidCallback onExpired;

  /// Called when the challenge or the script load fails.
  final VoidCallback onError;

  const TurnstileView({
    required this.siteKey,
    required this.onToken,
    required this.onExpired,
    required this.onError,
    super.key,
  });

  @override
  State<TurnstileView> createState() => _TurnstileViewState();
}

class _TurnstileViewState extends State<TurnstileView> {
  web.HTMLElement? _element;
  String? _widgetId;

  @override
  void dispose() {
    final id = _widgetId;
    if (id != null && globalContext.has('turnstile')) {
      try {
        (globalContext['turnstile'] as JSObject)
            .callMethod<JSAny?>('remove'.toJS, id.toJS);
      } catch (_) {
        // Already torn down.
      }
    }
    super.dispose();
  }

  Future<void> _render() async {
    try {
      await _loadScript();
    } catch (_) {
      if (mounted) widget.onError();
      return;
    }
    // Turnstile measures its container, so wait until the platform view
    // has attached the element to the page (a few frames at most).
    for (var i = 0; i < 120; i++) {
      if (!mounted || (_element?.isConnected ?? false)) break;
      await Future<void>.delayed(const Duration(milliseconds: 16));
    }
    final element = _element;
    if (!mounted || element == null) return;

    final options = JSObject()
      ..['sitekey'] = widget.siteKey.toJS
      ..['callback'] = ((JSString token) {
        if (mounted) widget.onToken(token.toDart);
      }).toJS
      ..['expired-callback'] = (() {
        if (mounted) widget.onExpired();
      }).toJS
      ..['error-callback'] = (() {
        if (mounted) widget.onError();
      }).toJS;
    final id = (globalContext['turnstile'] as JSObject)
        .callMethod<JSString?>('render'.toJS, element, options);
    _widgetId = id?.toDart;
  }

  @override
  Widget build(BuildContext context) {
    // 300x65 is Turnstile's managed-widget size.
    return SizedBox(
      width: 300,
      height: 65,
      child: HtmlElementView.fromTagName(
        tagName: 'div',
        onElementCreated: (Object element) {
          _element = element as web.HTMLElement;
          _render();
        },
      ),
    );
  }
}
