import 'package:doodhee/app/providers.dart';
import 'package:doodhee/app/theme.dart';
import 'package:doodhee/core/api.dart';
import 'package:doodhee/core/repository.dart';
import 'package:doodhee/features/screens.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

class WorkflowRepository extends MarketplaceRepository {
  final Map<String, dynamic> responses;
  final List<(String, String?, Json?)> mutations = [];

  WorkflowRepository(super.api, super.local, super.session, this.responses);

  @override
  Future<dynamic> read(
    String op, {
    String? id,
    Map<String, String>? query,
  }) async => responses[op] ?? <String, dynamic>{};

  @override
  Future<dynamic> mutate(
    String op, {
    String? id,
    Json? body,
    String? key,
  }) async {
    mutations.add((op, id, body));
    return <String, dynamic>{};
  }

  @override
  Future<List<Json>> pending() async => [];
}

Future<void> pumpWorkflow(
  WidgetTester tester, {
  required String role,
  required String section,
  required String id,
  required WorkflowRepository repository,
  required SessionRepository session,
}) async {
  await tester.binding.setSurfaceSize(const Size(900, 1400));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        sessionProvider.overrideWithValue(session),
        repositoryProvider.overrideWithValue(repository),
      ],
      child: MaterialApp(
        theme: doodheeTheme(),
        home: WorkspaceScreen(role: role, section: section, id: id),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Milkman looks up an existing phone before connecting customer', (
    tester,
  ) async {
    final api = ApiService();
    final session = SessionRepository(api)
      ..ready = true
      ..user = {'id': 'milkman-1', 'role': 'seller', 'is_enabled': true};
    final repository = WorkflowRepository(api, LocalStore(), session, {
      'seller_customers': {'items': [], 'next_cursor': ''},
      'seller_customer_lookup': {
        'exists': true,
        'id': 'customer-1',
        'full_name': 'Existing Aryan',
        'phone': '+919098056571',
        'already_connected': false,
      },
    });

    await tester.binding.setSurfaceSize(const Size(900, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          sessionProvider.overrideWithValue(session),
          repositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          theme: doodheeTheme(),
          home: const WorkspaceScreen(role: 'seller', section: 'customers'),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(FloatingActionButton));
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Phone number'),
      '+919098056571',
    );
    await tester.tap(find.text('Find customer'));
    await tester.pumpAndSettle();

    expect(find.text('Existing Aryan'), findsOneWidget);
    expect(find.textContaining('Existing customer found'), findsOneWidget);
    expect(find.text('Connect customer'), findsOneWidget);
    expect(find.widgetWithText(TextFormField, 'Full name'), findsNothing);
    expect(find.text('Delivery address'), findsOneWidget);
  });

  testWidgets('Milkman customer detail exposes pricing and daily delivery', (
    tester,
  ) async {
    final api = ApiService();
    final session = SessionRepository(api)
      ..ready = true
      ..user = {'id': 'milkman-1', 'role': 'seller', 'is_enabled': true};
    final repository = WorkflowRepository(api, LocalStore(), session, {
      'seller_customer': {
        'id': 'customer-1',
        'full_name': 'Meera Customer',
        'phone': '+919100000001',
        'delivery_address': '12 Green Street, Amritsar',
        'phone_verified_at': null,
      },
      'seller_products': {
        'items': [
          {
            'id': 'product-1',
            'name': 'Cow milk',
            'price': '62.00',
            'daily_capacity': '100.000',
            'is_available': true,
          },
        ],
      },
      'seller_customer_prices': {
        'items': [
          {'product_id': 'product-1', 'price': '59.00'},
        ],
      },
    });

    await pumpWorkflow(
      tester,
      role: 'seller',
      section: 'customers',
      id: 'customer-1',
      repository: repository,
      session: session,
    );

    expect(find.text('Meera Customer'), findsOneWidget);
    expect(find.textContaining('Waiting for first OTP login'), findsOneWidget);
    expect(find.textContaining('Customer ₹ 59.00'), findsOneWidget);
    expect(find.text('Record delivery'), findsOneWidget);
    expect(find.text('Customer price'), findsOneWidget);

    await tester.tap(find.text('Record delivery'));
    await tester.pumpAndSettle();
    expect(find.byType(BottomSheet), findsOneWidget);
    expect(find.text('Delivery date'), findsOneWidget);
    expect(find.text('Quantity in litres'), findsOneWidget);
    expect(find.text('Delivery-only price (optional)'), findsOneWidget);
    expect(find.text('Delivery address'), findsOneWidget);
  });

  testWidgets('Buyer sees ranking and edits the existing Milkman rating', (
    tester,
  ) async {
    final api = ApiService();
    final session = SessionRepository(api)
      ..ready = true
      ..user = {'id': 'customer-1', 'role': 'customer', 'is_enabled': true};
    final repository = WorkflowRepository(api, LocalStore(), session, {
      'products': {
        'items': [
          {
            'id': 'product-1',
            'name': 'Buffalo milk',
            'price': '75.00',
            'effective_price': '72.00',
            'price_source': 'customer',
          },
        ],
      },
      'discovery_detail': {
        'id': 'milkman-1',
        'full_name': 'Arjun Milkman',
        'rank': 2,
        'average_rating': '4.50',
        'rating_count': 18,
      },
      'rating_get': {'id': 'rating-1', 'rating': 4},
    });

    await pumpWorkflow(
      tester,
      role: 'buyer',
      section: 'sellers',
      id: 'milkman-1',
      repository: repository,
      session: session,
    );

    expect(find.text('Arjun Milkman'), findsOneWidget);
    expect(find.text('#2 · ⭐ 4.50 (18)'), findsOneWidget);
    expect(find.text('Customer price'), findsOneWidget);

    await tester.tap(find.byTooltip('5 / 5'));
    await tester.pumpAndSettle();
    expect(repository.mutations.last.$1, 'rating_set');
    expect(repository.mutations.last.$2, 'milkman-1');
    expect(repository.mutations.last.$3?['rating'], '5');
  });

  testWidgets('Buyer order detail exposes an extra milk request', (
    tester,
  ) async {
    final api = ApiService();
    final session = SessionRepository(api)
      ..ready = true
      ..user = {'id': 'customer-1', 'role': 'customer', 'is_enabled': true};
    final repository = WorkflowRepository(api, LocalStore(), session, {
      'order': {
        'id': 'order-1',
        'status': 'accepted',
        'service_date': '2026-11-15',
        'quantity': '1.000',
        'base_quantity': '1.000',
        'approved_extra_quantity': '0.000',
        'unit_price': '62.00',
        'price_source': 'default',
        'total_amount': '62.00',
        'delivery_address': '12 Green Street',
        'customer_note': '',
        'delivery_note': '',
      },
      'extra_request_get': {'order_id': 'order-1', 'status': 'none'},
    });

    await pumpWorkflow(
      tester,
      role: 'buyer',
      section: 'orders',
      id: 'order-1',
      repository: repository,
      session: session,
    );

    expect(find.text('Request extra milk'), findsOneWidget);
    expect(find.text('Price level: default'), findsOneWidget);
    await tester.tap(find.text('Request extra milk'));
    await tester.pumpAndSettle();
    expect(find.byType(BottomSheet), findsOneWidget);
    expect(find.text('Extra quantity in litres'), findsOneWidget);
  });
}
