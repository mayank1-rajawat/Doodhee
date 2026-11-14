import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:doodhee/core/api.dart';
import 'package:doodhee/core/repository.dart';
import 'package:doodhee/app/app.dart';
import 'package:doodhee/app/strings.dart';
import 'package:doodhee/app/theme.dart';
import 'package:doodhee/features/screens.dart';

void main() {
  test('Enabled accounts route directly to their role experience', () {
    final session = SessionRepository(ApiService());
    expect(roleRedirect(session, '/buyer/orders'), '/splash');
    session.ready = true;
    expect(roleRedirect(session, '/seller/orders'), '/auth');
    for (final state in ['pending', 'rejected', 'approved']) {
      session.user = {
        'role': 'seller',
        'is_enabled': true,
        'approval_status': state,
      };
      expect(roleRedirect(session, '/buyer/home'), '/seller/home');
    }
    session.user = {'role': 'customer', 'is_enabled': true};
    expect(roleRedirect(session, '/seller/orders'), '/buyer/home');
    expect(roleRedirect(session, '/buyer/orders'), null);
    session.user = {'role': 'customer', 'is_enabled': false};
    expect(roleRedirect(session, '/buyer/orders'), '/disabled');
  });
  test('All initial languages have status labels', () {
    for (final lang in ['en', 'hi', 'pa']) {
      expect(Strings(lang).t('delivered').isNotEmpty, true);
      expect(Strings(lang).t('not_delivered').isNotEmpty, true);
    }
  });
  testWidgets('Login presents buyer and seller choices and localized fields', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(theme: doodheeTheme(), home: const AuthScreen()),
      ),
    );
    expect(find.text('Doodhee'), findsOneWidget);
    expect(find.text('Buyer'), findsOneWidget);
    expect(find.text('Milkman'), findsOneWidget);
    expect(find.text('Phone number'), findsOneWidget);
    expect(find.text('Sign in'), findsOneWidget);
    expect(find.text('New account'), findsOneWidget);
    expect(find.text('Send login code'), findsOneWidget);
    expect(find.text('Full name'), findsNothing);

    await tester.tap(find.text('New account'));
    await tester.pumpAndSettle();
    expect(find.text('Full name'), findsOneWidget);
  });
  testWidgets('Delivery chip includes text as well as color', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: doodheeTheme(),
        home: const Scaffold(
          body: StatusChip('not_delivered', 'Not delivered'),
        ),
      ),
    );
    expect(find.text('Not delivered'), findsOneWidget);
    expect(find.byIcon(Icons.circle), findsOneWidget);
  });
}
