import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:connectivity_plus/connectivity_plus.dart';

import '../core/repository.dart';
import '../features/screens.dart';
import 'providers.dart';
import 'theme.dart';

String? roleRedirect(SessionRepository session, String location) {
  final target = session.destination;
  if (target == '/splash') return location == target ? null : target;
  if (target == '/auth' || target == '/disabled') {
    return location == target ? null : target;
  }
  final prefix = target.startsWith('/buyer') ? '/buyer' : '/seller';
  return location.startsWith(prefix) ? null : target;
}

class DoodheeApp extends ConsumerStatefulWidget {
  const DoodheeApp({super.key});
  @override
  ConsumerState<DoodheeApp> createState() => _DoodheeAppState();
}

class _DoodheeAppState extends ConsumerState<DoodheeApp>
    with WidgetsBindingObserver {
  late final GoRouter router;
  StreamSubscription<List<ConnectivityResult>>? connectivity;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    final session = ref.read(sessionProvider);
    router = GoRouter(
      initialLocation: '/splash',
      refreshListenable: session,
      redirect: (context, state) => roleRedirect(session, state.uri.path),
      routes: [
        GoRoute(
          path: '/splash',
          builder: (_, state) =>
              const Scaffold(body: Center(child: CircularProgressIndicator())),
        ),
        GoRoute(path: '/auth', builder: (_, state) => const AuthScreen()),
        GoRoute(path: '/disabled', builder: (_, state) => const StatusScreen()),
        GoRoute(
          path: '/buyer/:section',
          builder: (_, state) => WorkspaceScreen(
            role: 'buyer',
            section: state.pathParameters['section']!,
          ),
        ),
        GoRoute(
          path: '/buyer/:section/:id',
          builder: (_, state) => WorkspaceScreen(
            role: 'buyer',
            section: state.pathParameters['section']!,
            id: state.pathParameters['id'],
          ),
        ),
        GoRoute(
          path: '/seller/:section',
          builder: (_, state) => WorkspaceScreen(
            role: 'seller',
            section: state.pathParameters['section']!,
          ),
        ),
        GoRoute(
          path: '/seller/:section/:id',
          builder: (_, state) => WorkspaceScreen(
            role: 'seller',
            section: state.pathParameters['section']!,
            id: state.pathParameters['id'],
          ),
        ),
      ],
    );
    unawaited(session.restore());
    connectivity = Connectivity().onConnectivityChanged.listen((values) {
      if (!values.contains(ConnectivityResult.none)) unawaited(_sync());
    });
  }

  Future<void> _sync() async {
    if (ref.read(sessionProvider).user?['role'] == 'seller') {
      try {
        await ref.read(repositoryProvider).sync();
      } catch (_) {}
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) unawaited(_sync());
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    connectivity?.cancel();
    router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp.router(
    title: 'Doodhee',
    debugShowCheckedModeBanner: false,
    theme: doodheeTheme(),
    routerConfig: router,
  );
}
