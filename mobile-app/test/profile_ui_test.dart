import 'dart:convert';

import 'package:doodhee/app/providers.dart';
import 'package:doodhee/app/theme.dart';
import 'package:doodhee/core/api.dart';
import 'package:doodhee/core/repository.dart';
import 'package:doodhee/features/screens.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:material_ui/material_ui.dart';

class ProfileRepository extends MarketplaceRepository {
  final Json profile;
  ProfileRepository(super.api, super.local, super.session, this.profile);

  @override
  Future<dynamic> read(
    String op, {
    String? id,
    Map<String, String>? query,
  }) async => switch (op) {
    'me' => profile,
    'addresses' => {
      'items': [
        {
          'id': 'home-id',
          'label': 'Home',
          'address_line': '12 Green Street',
          'city': 'Amritsar',
          'pincode': '143001',
          'is_default': true,
        },
        {
          'id': 'work-id',
          'label': 'Work',
          'address_line': '45 Market Road',
          'city': 'Amritsar',
          'pincode': '143001',
          'is_default': false,
        },
      ],
      'next_cursor': '',
    },
    _ => <String, dynamic>{},
  };
}

void main() {
  testWidgets('Buyer profile shows full account details and saved addresses', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(800, 1200));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final user = <String, dynamic>{
      'id': 'buyer-id',
      'full_name': 'Aman Buyer',
      'phone': '+919876543210',
      'role': 'customer',
      'approval_status': 'approved',
      'is_enabled': true,
    };
    final api = ApiService(
      client: MockClient((request) async {
        final data = switch (request.url.path) {
          '/v1/me' => user,
          '/v1/addresses' => {
            'items': [
              {
                'id': 'home-id',
                'label': 'Home',
                'address_line': '12 Green Street',
                'city': 'Amritsar',
                'pincode': '143001',
                'is_default': true,
              },
              {
                'id': 'work-id',
                'label': 'Work',
                'address_line': '45 Market Road',
                'city': 'Amritsar',
                'pincode': '143001',
                'is_default': false,
              },
            ],
            'next_cursor': '',
          },
          _ => <String, dynamic>{},
        };
        return http.Response(jsonEncode({'data': data}), 200);
      }),
    );
    final session = SessionRepository(api)
      ..user = user
      ..ready = true;
    final repository = ProfileRepository(api, LocalStore(), session, user);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          apiProvider.overrideWithValue(api),
          sessionProvider.overrideWithValue(session),
          repositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          theme: doodheeTheme(),
          home: const WorkspaceScreen(role: 'buyer', section: 'profile'),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Aman Buyer'), findsOneWidget);
    expect(find.text('Verified phone'), findsOneWidget);
    expect(find.text('Saved addresses'), findsOneWidget);
    expect(find.text('Home'), findsAtLeastNWidgets(1));
    expect(find.text('Work'), findsOneWidget);
    expect(find.text('Default'), findsOneWidget);

    final addAddress = find.widgetWithText(FilledButton, 'Add address');
    await tester.tap(addAddress);
    await tester.pumpAndSettle();

    expect(find.byType(BottomSheet), findsOneWidget);
    expect(find.byType(AlertDialog), findsNothing);
    expect(find.text('Label (Home, Work…)'), findsOneWidget);
    expect(find.text('City'), findsOneWidget);
    expect(find.text('Pincode'), findsOneWidget);
  });
}
