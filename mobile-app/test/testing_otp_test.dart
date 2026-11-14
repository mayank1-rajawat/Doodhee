import 'dart:convert';

import 'package:doodhee/app/providers.dart';
import 'package:doodhee/app/theme.dart';
import 'package:doodhee/core/api.dart';
import 'package:doodhee/features/screens.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final scenario in [
    (
      name: 'Buyer login OTP hidden by default',
      role: 'customer',
      enabled: false,
      register: false,
    ),
    (
      name: 'Buyer registration OTP visible for testing',
      role: 'customer',
      enabled: true,
      register: true,
    ),
    (
      name: 'Milkman registration OTP visible for testing',
      role: 'seller',
      enabled: true,
      register: true,
    ),
  ]) {
    testWidgets(scenario.name, (tester) async {
      Json? submitted;
      final api = ApiService(
        client: MockClient((request) async {
          submitted = jsonDecode(request.body) as Json;
          return http.Response(
            jsonEncode({
              'data': {
                'challenge_id': 'test-challenge',
                'expires_in': 300,
                if (scenario.enabled) 'testing_otp': '012345',
              },
            }),
            200,
          );
        }),
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [apiProvider.overrideWithValue(api)],
          child: MaterialApp(theme: doodheeTheme(), home: const AuthScreen()),
        ),
      );
      if (scenario.role == 'seller') {
        await tester.tap(find.text('Milkman'));
        await tester.pumpAndSettle();
      }
      if (scenario.register) {
        await tester.tap(find.text('New account'));
        await tester.pumpAndSettle();
        await tester.enterText(
          find.widgetWithText(TextFormField, 'Full name'),
          'Test user',
        );
      }
      await tester.enterText(
        find.widgetWithText(TextFormField, 'Phone number'),
        '+15555550123',
      );
      final submit = find.widgetWithText(
        FilledButton,
        scenario.register ? 'Create account' : 'Send login code',
      );
      await tester.ensureVisible(submit);
      await tester.pumpAndSettle();
      await tester.tap(submit);
      await tester.pumpAndSettle();
      expect(submitted?['role'], scenario.role);
      expect(submitted?['mode'], scenario.register ? 'register' : 'login');
      expect(find.text('Verification code'), findsOneWidget);
      if (scenario.enabled) {
        expect(find.text('Testing OTP'), findsOneWidget);
        expect(
          tester.widget<SelectableText>(find.byType(SelectableText)).data,
          '012345',
        );
        expect(
          tester
              .widget<TextFormField>(find.byType(TextFormField))
              .controller
              ?.text,
          isEmpty,
        );
        await tester.tap(find.byTooltip('Change phone'));
        await tester.pumpAndSettle();
        expect(find.text('Testing OTP'), findsNothing);
        expect(find.byType(SelectableText), findsNothing);
      } else {
        expect(find.text('Testing OTP'), findsNothing);
        expect(find.byType(SelectableText), findsNothing);
      }
    });
  }
}
