import 'dart:async';

import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:uuid/uuid.dart';

import '../app/providers.dart';
import '../app/strings.dart';
import '../core/api.dart';
import '../core/repository.dart';

String value(dynamic v) => v == null ? '—' : v.toString();
String day(DateTime d) => d.toIso8601String().substring(0, 10);
String addressText(Map<String, dynamic> address) => [
  address['address_line'],
  address['city'],
  address['pincode'],
].where((part) => part != null && part.toString().trim().isNotEmpty).join(', ');
Widget field(
  String label,
  TextEditingController controller, {
  bool number = false,
  bool required = true,
}) => Padding(
  padding: const EdgeInsets.only(bottom: 16),
  child: TextFormField(
    controller: controller,
    keyboardType: number
        ? const TextInputType.numberWithOptions(decimal: true)
        : TextInputType.text,
    decoration: InputDecoration(labelText: label),
    validator: (v) =>
        required && (v == null || v.trim().isEmpty) ? 'Required' : null,
  ),
);

class StatusChip extends StatelessWidget {
  final String status, label;
  const StatusChip(this.status, this.label, {super.key});
  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      'delivered' || 'approved' => const Color(0xff15803D),
      'not_delivered' || 'rejected' => const Color(0xffB91C1C),
      'pending' => const Color(0xffB45309),
      'out_for_delivery' => const Color(0xff1D4ED8),
      _ => const Color(0xff64748B),
    };
    return Chip(
      label: Text(label),
      avatar: Icon(Icons.circle, size: 9, color: color),
      backgroundColor: color.withValues(alpha: .08),
      side: BorderSide.none,
      labelStyle: TextStyle(color: color, fontSize: 12),
    );
  }
}

class AuthScreen extends ConsumerStatefulWidget {
  const AuthScreen({super.key});
  @override
  ConsumerState<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends ConsumerState<AuthScreen> {
  late final AuthViewModel auth;
  @override
  void initState() {
    super.initState();
    auth = AuthViewModel(ref.read(sessionProvider));
  }

  final phone = TextEditingController(),
      name = TextEditingController(),
      code = TextEditingController();
  String role = 'customer', challenge = '', error = '';
  bool busy = false, registering = false;
  @override
  void dispose() {
    phone.dispose();
    name.dispose();
    code.dispose();
    super.dispose();
  }

  Future<void> submit() async {
    setState(() {
      busy = true;
      error = '';
    });
    try {
      if (challenge.isEmpty) {
        if (!RegExp(r'^\+[1-9][0-9]{9,15}$').hasMatch(phone.text.trim())) {
          throw ApiFailure(
            422,
            'VALIDATION_ERROR',
            'Enter your phone number with country code, for example +91.',
          );
        }
        if (registering && name.text.trim().isEmpty) {
          throw ApiFailure(422, 'VALIDATION_ERROR', 'Enter your full name.');
        }
        final id = await auth.requestCode(
          phone.text,
          role,
          mode: registering ? 'register' : 'login',
        );
        if (mounted) setState(() => challenge = id);
      } else {
        await auth.verify(<String, dynamic>{
          'challenge_id': challenge,
          'code': code.text.trim(),
          if (registering) 'full_name': name.text.trim(),
        });
      }
    } catch (e) {
      if (mounted) setState(() => error = e.toString());
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  void resetCode() => setState(() {
    challenge = '';
    error = '';
    auth.testingOtp = null;
    code.clear();
  });

  void changeMode(bool value) => setState(() {
    registering = value;
    challenge = '';
    error = '';
    auth.testingOtp = null;
    code.clear();
  });

  Widget accountType(
    BuildContext context,
    String value,
    IconData icon,
    String title,
    String subtitle,
  ) {
    final selected = role == value;
    return Expanded(
      child: Semantics(
        selected: selected,
        button: true,
        child: InkWell(
          borderRadius: BorderRadius.circular(16),
          onTap: busy ? null : () => setState(() => role = value),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 180),
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: selected
                  ? const Color(0xffECFDF3)
                  : const Color(0xffF8FAF9),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(
                color: selected
                    ? const Color(0xff15803D)
                    : const Color(0xffD7E2DA),
                width: selected ? 1.5 : 1,
              ),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(icon, color: const Color(0xff166534)),
                const SizedBox(height: 10),
                Text(title, style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 2),
                Text(subtitle, style: Theme.of(context).textTheme.bodySmall),
              ],
            ),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = Strings(ref.watch(languageProvider));
    const buildLabel = String.fromEnvironment('BUILD_LABEL');
    return Scaffold(
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 28),
          children: [
            Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 460),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Row(
                      children: [
                        ClipRRect(
                          borderRadius: BorderRadius.circular(15),
                          child: Image.asset(
                            'assets/images/doodhee-icon.png',
                            width: 48,
                            height: 48,
                            fit: BoxFit.cover,
                            semanticLabel: 'Doodhee',
                          ),
                        ),
                        const SizedBox(width: 12),
                        const Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                'Doodhee',
                                style: TextStyle(
                                  fontSize: 25,
                                  height: 1,
                                  fontWeight: FontWeight.w800,
                                  color: Color(0xff14532D),
                                ),
                              ),
                              SizedBox(height: 5),
                              Text('Fresh milk, delivered locally'),
                            ],
                          ),
                        ),
                        languagePicker(ref),
                      ],
                    ),
                    const SizedBox(height: 32),
                    Container(
                      padding: const EdgeInsets.all(22),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(24),
                        border: Border.all(color: const Color(0xffD7E2DA)),
                        boxShadow: const [
                          BoxShadow(
                            color: Color(0x0D14532D),
                            blurRadius: 24,
                            offset: Offset(0, 8),
                          ),
                        ],
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          if (challenge.isEmpty) ...[
                            Text(
                              s.t(
                                registering ? 'create_account' : 'welcome_back',
                              ),
                              style: Theme.of(context).textTheme.headlineSmall,
                            ),
                            const SizedBox(height: 7),
                            Text(
                              s.t(registering ? 'register_help' : 'login_help'),
                            ),
                            const SizedBox(height: 22),
                            SegmentedButton<String>(
                              segments: [
                                ButtonSegment(
                                  value: 'login',
                                  label: Text(s.t('login')),
                                  icon: const Icon(Icons.login_rounded),
                                ),
                                ButtonSegment(
                                  value: 'register',
                                  label: Text(s.t('register')),
                                  icon: const Icon(Icons.person_add_alt_1),
                                ),
                              ],
                              selected: {registering ? 'register' : 'login'},
                              onSelectionChanged: busy
                                  ? null
                                  : (selection) => changeMode(
                                      selection.first == 'register',
                                    ),
                            ),
                            const SizedBox(height: 24),
                            Text(
                              s.t('account_type'),
                              style: Theme.of(context).textTheme.titleSmall,
                            ),
                            const SizedBox(height: 10),
                            Row(
                              children: [
                                accountType(
                                  context,
                                  'customer',
                                  Icons.home_outlined,
                                  s.t('buyer'),
                                  s.t('buyer_help'),
                                ),
                                const SizedBox(width: 10),
                                accountType(
                                  context,
                                  'seller',
                                  Icons.storefront_outlined,
                                  s.t('seller'),
                                  s.t('seller_help'),
                                ),
                              ],
                            ),
                            const SizedBox(height: 22),
                            if (registering)
                              TextFormField(
                                controller: name,
                                textCapitalization: TextCapitalization.words,
                                autofillHints: const [AutofillHints.name],
                                decoration: InputDecoration(
                                  labelText: s.t('name'),
                                  prefixIcon: const Icon(Icons.person_outline),
                                ),
                              ),
                            if (registering) const SizedBox(height: 14),
                            TextFormField(
                              controller: phone,
                              keyboardType: TextInputType.phone,
                              autofillHints: const [
                                AutofillHints.telephoneNumber,
                              ],
                              decoration: InputDecoration(
                                labelText: s.t('phone'),
                                hintText: '+91 98765 43210',
                                prefixIcon: const Icon(Icons.phone_outlined),
                                helperText: s.t('country_code_help'),
                              ),
                              onFieldSubmitted: busy ? null : (_) => submit(),
                            ),
                          ] else ...[
                            Row(
                              children: [
                                IconButton.filledTonal(
                                  tooltip: s.t('Change phone'),
                                  onPressed: busy ? null : resetCode,
                                  icon: const Icon(Icons.arrow_back),
                                ),
                                const SizedBox(width: 14),
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
                                    children: [
                                      Text(
                                        s.t('enter_code'),
                                        style: Theme.of(context)
                                            .textTheme
                                            .headlineSmall,
                                      ),
                                      Text(
                                        '${phone.text.trim()} · ${s.t(role == 'customer' ? 'buyer' : 'seller')}',
                                      ),
                                    ],
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 22),
                            TextFormField(
                              controller: code,
                              autofocus: true,
                              keyboardType: TextInputType.number,
                              textAlign: TextAlign.center,
                              maxLength: 6,
                              style: const TextStyle(
                                fontSize: 26,
                                fontWeight: FontWeight.w700,
                                letterSpacing: 8,
                              ),
                              decoration: InputDecoration(
                                labelText: s.t('code'),
                                counterText: '',
                              ),
                              onFieldSubmitted: busy ? null : (_) => submit(),
                            ),
                            const SizedBox(height: 14),
                          ],
                          if (challenge.isNotEmpty && auth.testingOtp != null)
                            Semantics(
                              liveRegion: true,
                              child: Container(
                                margin: const EdgeInsets.only(bottom: 16),
                                padding: const EdgeInsets.all(16),
                                decoration: BoxDecoration(
                                  color: const Color(0xffECFDF3),
                                  borderRadius: BorderRadius.circular(16),
                                  border: Border.all(
                                    color: const Color(0xffBBF7D0),
                                  ),
                                ),
                                child: Row(
                                  children: [
                                    const Icon(
                                      Icons.science_outlined,
                                      color: Color(0xff166534),
                                    ),
                                    const SizedBox(width: 12),
                                    Expanded(
                                      child: Column(
                                        crossAxisAlignment:
                                            CrossAxisAlignment.start,
                                        children: [
                                          Text(s.t('testing_otp')),
                                          SelectableText(
                                            auth.testingOtp!,
                                            style: const TextStyle(
                                              fontSize: 24,
                                              fontWeight: FontWeight.w800,
                                              letterSpacing: 4,
                                              color: Color(0xff14532D),
                                            ),
                                          ),
                                          Text(s.t('testing_only')),
                                        ],
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          if (error.isNotEmpty)
                            Container(
                              margin: const EdgeInsets.only(bottom: 16),
                              padding: const EdgeInsets.all(14),
                              decoration: BoxDecoration(
                                color: const Color(0xffFEF2F2),
                                borderRadius: BorderRadius.circular(14),
                              ),
                              child: Row(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  const Icon(
                                    Icons.error_outline,
                                    color: Color(0xffB91C1C),
                                  ),
                                  const SizedBox(width: 10),
                                  Expanded(
                                    child: Text(
                                      error,
                                      style: const TextStyle(
                                        color: Color(0xff991B1B),
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          FilledButton.icon(
                            onPressed: busy ? null : submit,
                            icon: busy
                                ? const SizedBox.square(
                                    dimension: 18,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                    ),
                                  )
                                : Icon(
                                    challenge.isEmpty
                                        ? Icons.arrow_forward
                                        : Icons.verified_outlined,
                                  ),
                            label: Text(
                              challenge.isNotEmpty
                                  ? s.t('verify_continue')
                                  : s.t(
                                      registering
                                          ? 'create_send_code'
                                          : 'send_login_code',
                                    ),
                            ),
                          ),
                          if (challenge.isEmpty) ...[
                            const SizedBox(height: 14),
                            Text(
                              s.t('otp_security_help'),
                              textAlign: TextAlign.center,
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                          ],
                        ],
                      ),
                    ),
                    if (buildLabel.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 18),
                        child: Text(
                          buildLabel,
                          textAlign: TextAlign.center,
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

Widget languagePicker(WidgetRef ref) => DropdownButton<String>(
  value: ref.watch(languageProvider),
  items: const [
    DropdownMenuItem(value: 'en', child: Text('English')),
    DropdownMenuItem(value: 'hi', child: Text('हिन्दी')),
    DropdownMenuItem(value: 'pa', child: Text('ਪੰਜਾਬੀ')),
  ],
  onChanged: (v) => ref.read(languageProvider.notifier).change(v!),
);

class StatusScreen extends ConsumerWidget {
  const StatusScreen({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final session = ref.watch(sessionProvider);
    final s = Strings(ref.watch(languageProvider));
    return Scaffold(
      appBar: AppBar(title: const Text('Doodhee')),
      body: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Icon(
              Icons.verified_user_outlined,
              size: 64,
              color: Color(0xff166534),
            ),
            const SizedBox(height: 24),
            Text(
              s.t('Account disabled'),
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 16),
            Text(s.t('Contact the administrator to restore access.')),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: () async {
                await ref.read(localProvider).clear();
                await session.invalidate();
              },
              child: Text(s.t('logout')),
            ),
          ],
        ),
      ),
    );
  }
}

class WorkspaceScreen extends ConsumerStatefulWidget {
  final String role, section;
  final String? id;
  const WorkspaceScreen({
    super.key,
    required this.role,
    required this.section,
    this.id,
  });
  @override
  ConsumerState<WorkspaceScreen> createState() => _WorkspaceScreenState();
}

class _WorkspaceScreenState extends ConsumerState<WorkspaceScreen> {
  late DataViewModel vm;
  String from = day(DateTime(DateTime.now().year, DateTime.now().month)),
      to = day(DateTime.now());
  String customerSearch = '';
  List<Json> queued = [];
  bool get seller => widget.role == 'seller';
  Strings get s => Strings(ref.read(languageProvider));
  @override
  void initState() {
    super.initState();
    vm = DataViewModel(ref.read(repositoryProvider));
    if (widget.section == 'orders') {
      to = day(DateTime.now().add(const Duration(days: 90)));
    }
    unawaited(load());
  }

  @override
  void didUpdateWidget(covariant WorkspaceScreen old) {
    super.didUpdateWidget(old);
    if (old.section != widget.section || old.id != widget.id) {
      vm.data = null;
      to = day(
        widget.section == 'orders'
            ? DateTime.now().add(const Duration(days: 90))
            : DateTime.now(),
      );
      unawaited(load());
    }
  }

  @override
  void dispose() {
    vm.dispose();
    super.dispose();
  }

  Future<void> load() async {
    final section = widget.section;
    String op;
    Map<String, String>? query;
    if (section == 'profile') {
      op = 'me';
    } else if (section == 'summary') {
      op = seller ? 'seller_summary' : 'summary';
      query = {'from': from, 'to': to};
    } else if (section == 'products') {
      op = widget.id == null
          ? 'seller_products'
          : seller
          ? 'seller_product'
          : 'product';
    } else if (section == 'sellers') {
      op = 'products';
    } else if (section == 'customers') {
      op = widget.id == null ? 'seller_customers' : 'seller_customer';
      if (widget.id == null && customerSearch.isNotEmpty) {
        query = {'search': customerSearch};
      }
    } else if (section == 'requests') {
      op = 'seller_extra_requests';
    } else if (section == 'orders') {
      op = seller
          ? (widget.id == null ? 'seller_orders' : 'seller_order')
          : (widget.id == null ? 'orders' : 'order');
      if (widget.id == null) query = {'from': from, 'to': to};
    } else {
      op = seller ? 'seller_orders' : 'sellers';
      if (seller) {
        query = {'service_date': day(DateTime.now())};
      } else {
        final pin = ref.read(sessionProvider).user?['pincode'];
        if (pin != null && pin != '') query = {'pincode': pin.toString()};
      }
    }
    await vm.load(op, id: widget.id, query: query);
    if (section == 'customers' && widget.id != null && vm.data is Map) {
      final products = await vm.repository.read('seller_products') as Json;
      final prices = await vm.repository.read(
        'seller_customer_prices',
        id: widget.id,
      ) as Json;
      vm.data = {
        ...Map<String, dynamic>.from(vm.data as Map),
        'products': products['items'] as List? ?? [],
        'prices': prices['items'] as List? ?? [],
      };
      vm.notifyListeners();
    }
    if (section == 'sellers' && widget.id != null && vm.data is Map) {
      final detail =
          await vm.repository.read('discovery_detail', id: widget.id) as Json;
      final rating =
          await vm.repository.read('rating_get', id: widget.id) as Json;
      vm.data = {
        ...Map<String, dynamic>.from(vm.data as Map),
        'seller': detail,
        'rating': rating,
      };
      vm.notifyListeners();
    }
    if (section == 'orders' && widget.id != null && vm.data is Map) {
      final extra =
          await vm.repository.read('extra_request_get', id: widget.id) as Json;
      vm.data = {
        ...Map<String, dynamic>.from(vm.data as Map),
        'extra_request': extra,
      };
      vm.notifyListeners();
    }
    if (section == 'profile' && !seller && vm.data is Map) {
      try {
        final page = await vm.repository.read('addresses') as Json;
        vm.data = {
          ...Map<String, dynamic>.from(vm.data as Map),
          'addresses': page['items'] as List? ?? [],
        };
        vm.notifyListeners();
      } catch (_) {
        final cached = await vm.repository.cached('addresses');
        if (cached is Map) {
          vm.data = {
            ...Map<String, dynamic>.from(vm.data as Map),
            'addresses': cached['items'] as List? ?? [],
          };
          vm.notifyListeners();
        }
      }
    }
    if (seller) {
      queued = await ref.read(repositoryProvider).pending();
      if (mounted) setState(() {});
    }
  }

  Future<void> mutate(String op, {String? id, Json? body}) async {
    final ok = await vm.action(() async {
      await vm.repository.mutate(op, id: id, body: body);
    });
    if (ok) await load();
  }

  Future<void> form(
    String title,
    Map<String, String> defaults,
    Future<void> Function(Json) save,
  ) async {
    final controllers = defaults.map(
      (k, v) => MapEntry(k, TextEditingController(text: v)),
    );
    final key = GlobalKey<FormState>();
    String? error;
    bool saving = false;
    if (!mounted) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (context) => StatefulBuilder(
        builder: (context, setSheet) => Padding(
          padding: EdgeInsets.fromLTRB(
            24,
            0,
            24,
            MediaQuery.viewInsetsOf(context).bottom + 24,
          ),
          child: SingleChildScrollView(
            child: Form(
              key: key,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    s.t(title),
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    s.t('Complete the details below.'),
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 22),
                  for (final entry in controllers.entries)
                    if (entry.key == 'reason_code')
                      Padding(
                        padding: const EdgeInsets.only(bottom: 16),
                        child: DropdownButtonFormField<String>(
                          initialValue: entry.value.text,
                          decoration: InputDecoration(labelText: s.t('reason')),
                          items: [
                            for (final code in [
                              'customer_unavailable',
                              'seller_unavailable',
                              'insufficient_milk',
                              'address_issue',
                              'route_or_weather_issue',
                              'cancelled_before_delivery',
                              'other',
                            ])
                              DropdownMenuItem(
                                value: code,
                                child: Text(code.replaceAll('_', ' ')),
                              ),
                          ],
                          onChanged: (v) => entry.value.text = v!,
                        ),
                      )
                    else
                      field(
                        s.t(entry.key.replaceAll('_', ' ')),
                        entry.value,
                        number: [
                          'price',
                          'unit_price',
                          'daily_capacity',
                          'quantity',
                          'extra_quantity',
                          'pincode',
                        ].contains(entry.key),
                        required: ![
                          'customer_note',
                          'note',
                          'unit_price',
                          'decision_note',
                        ].contains(entry.key),
                      ),
                  if (error != null)
                    Container(
                      margin: const EdgeInsets.only(bottom: 16),
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: const Color(0xffFEF2F2),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        error!,
                        style: const TextStyle(color: Color(0xffB91C1C)),
                      ),
                    ),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton(
                          onPressed: saving
                              ? null
                              : () => Navigator.pop(context),
                          child: Text(s.t('Cancel')),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: FilledButton(
                          onPressed: saving
                              ? null
                              : () async {
                                  if (!key.currentState!.validate()) return;
                                  setSheet(() => saving = true);
                                  try {
                                    await save(
                                      controllers.map(
                                        (k, v) => MapEntry(k, v.text.trim()),
                                      ),
                                    );
                                    if (context.mounted) Navigator.pop(context);
                                  } catch (e) {
                                    setSheet(() => error = e.toString());
                                  } finally {
                                    if (context.mounted) {
                                      setSheet(() => saving = false);
                                    }
                                  }
                                },
                          child: Text(saving ? '…' : s.t('save')),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    for (final c in controllers.values) {
      c.dispose();
    }
    if (mounted) await load();
  }

  Future<void> productForm([Json? item]) => form(
    item == null ? 'Add milk product' : 'Edit product',
    {
      'name': item?['name']?.toString() ?? '',
      'price': item?['price']?.toString() ?? '',
      'daily_capacity': item?['daily_capacity']?.toString() ?? '',
    },
    (body) async {
      await vm.repository.mutate(
        item == null ? 'seller_product_create' : 'seller_product_update',
        id: item?['id'] as String?,
        body: body,
      );
    },
  );
  Future<void> customerForm() async {
    final phone = TextEditingController(text: '+91');
    final name = TextEditingController();
    final address = TextEditingController();
    final key = GlobalKey<FormState>();
    Json? lookup;
    String? error;
    var checking = false;
    var saving = false;
    if (!mounted) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (sheetContext) => StatefulBuilder(
        builder: (sheetContext, setSheet) => Padding(
          padding: EdgeInsets.fromLTRB(
            24,
            0,
            24,
            MediaQuery.viewInsetsOf(sheetContext).bottom + 24,
          ),
          child: SingleChildScrollView(
            child: Form(
              key: key,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    s.t('add_customer'),
                    style: Theme.of(sheetContext).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 8),
                  Text(s.t('find_customer_help')),
                  const SizedBox(height: 22),
                  TextFormField(
                    controller: phone,
                    keyboardType: TextInputType.phone,
                    decoration: InputDecoration(
                      labelText: s.t('phone'),
                      helperText: s.t('country_code_help'),
                      prefixIcon: const Icon(Icons.phone_outlined),
                    ),
                    onChanged: (_) => setSheet(() {
                      lookup = null;
                      error = null;
                    }),
                    validator: (value) => value == null || value.trim().isEmpty
                        ? s.t('Required')
                        : null,
                  ),
                  const SizedBox(height: 16),
                  OutlinedButton.icon(
                    onPressed: checking || saving
                        ? null
                        : () async {
                            if (phone.text.trim().isEmpty) return;
                            setSheet(() {
                              checking = true;
                              error = null;
                            });
                            try {
                              final result = await vm.repository.read(
                                'seller_customer_lookup',
                                query: {'phone': phone.text.trim()},
                              );
                              setSheet(() {
                                lookup = Map<String, dynamic>.from(
                                  result as Map,
                                );
                                if (lookup!['delivery_address'] != null) {
                                  address.text = lookup!['delivery_address']
                                      .toString();
                                }
                              });
                            } catch (e) {
                              setSheet(() => error = e.toString());
                            } finally {
                              if (sheetContext.mounted) {
                                setSheet(() => checking = false);
                              }
                            }
                          },
                    icon: const Icon(Icons.search),
                    label: Text(checking ? '…' : s.t('find_customer')),
                  ),
                  if (lookup != null) ...[
                    const SizedBox(height: 18),
                    if (lookup!['exists'] == true)
                      Card(
                        color: const Color(0xffECFDF3),
                        child: ListTile(
                          leading: const Icon(
                            Icons.person_search_outlined,
                            color: Color(0xff166534),
                          ),
                          title: Text(value(lookup!['full_name'])),
                          subtitle: Text(
                            '${lookup!['phone']}\n${lookup!['already_connected'] == true ? s.t('already_connected') : s.t('existing_customer_found')}',
                          ),
                        ),
                      )
                    else ...[
                      Text(
                        s.t('new_customer_help'),
                        style: Theme.of(sheetContext).textTheme.bodyMedium,
                      ),
                      const SizedBox(height: 16),
                      field(s.t('name'), name),
                    ],
                    field(s.t('address'), address),
                  ],
                  if (error != null)
                    Container(
                      margin: const EdgeInsets.only(bottom: 16),
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: const Color(0xffFEF2F2),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        error!,
                        style: const TextStyle(color: Color(0xffB91C1C)),
                      ),
                    ),
                  if (lookup != null)
                    FilledButton(
                      onPressed: saving
                          ? null
                          : () async {
                              if (!key.currentState!.validate()) return;
                              setSheet(() {
                                saving = true;
                                error = null;
                              });
                              try {
                                final customer = await vm.repository.mutate(
                                  'seller_customer_create',
                                  body: {
                                    'full_name': lookup!['exists'] == true
                                        ? lookup!['full_name'].toString()
                                        : name.text.trim(),
                                    'phone': phone.text.trim(),
                                    'delivery_address': address.text.trim(),
                                  },
                                ) as Json;
                                if (!sheetContext.mounted || !mounted) return;
                                Navigator.pop(sheetContext);
                                context.go(
                                  '/seller/customers/${customer['id']}',
                                );
                              } catch (e) {
                                setSheet(() => error = e.toString());
                              } finally {
                                if (sheetContext.mounted) {
                                  setSheet(() => saving = false);
                                }
                              }
                            },
                      child: Text(
                        saving
                            ? '…'
                            : s.t(
                                lookup!['exists'] == true
                                    ? 'connect_customer'
                                    : 'create_customer',
                              ),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    phone.dispose();
    name.dispose();
    address.dispose();
  }

  Future<void> customerPriceForm(Json customer, Json product) => form(
    'customer_price',
    {'price': product['customer_price']?.toString() ?? ''},
    (body) => vm.repository.mutate(
      'seller_customer_price_set',
      id: customer['id'] as String,
      body: {'product_id': product['id'], 'price': body['price']},
    ),
  );

  Future<void> directDeliveryForm(Json customer, Json product) => form(
    'record_delivery',
    {
      'service_date': day(DateTime.now()),
      'quantity': '1.000',
      'unit_price': '',
      'delivery_address': customer['delivery_address']?.toString() ?? '',
      'note': '',
    },
    (body) => vm.repository.mutate(
      'seller_delivery_create',
      body: {
        ...body,
        'customer_id': customer['id'],
        'product_id': product['id'],
      },
    ),
  );

  Future<void> extraMilkForm(Json order) => form(
    'extra_milk',
    {'extra_quantity': '0.500'},
    (body) => vm.repository.mutate(
      'extra_request_create',
      id: order['id'] as String,
      body: body,
    ),
  );
  Future<void> orderForm(Json product) async {
    List<Json> addresses;
    try {
      final page = await vm.repository.read('addresses') as Json;
      addresses = (page['items'] as List? ?? [])
          .map((item) => Map<String, dynamic>.from(item as Map))
          .toList();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
      return;
    }
    if (addresses.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(s.t('no_addresses')),
            action: SnackBarAction(
              label: s.t('add_address'),
              onPressed: () => context.go('/buyer/profile'),
            ),
          ),
        );
      }
      return;
    }
    final date = TextEditingController(
      text: day(DateTime.now().add(const Duration(days: 1))),
    );
    final quantity = TextEditingController(text: '1.000');
    final note = TextEditingController();
    final formKey = GlobalKey<FormState>();
    var selected = addresses
        .firstWhere(
          (address) => address['is_default'] == true,
          orElse: () => addresses.first,
        )['id']
        .toString();
    var saving = false;
    String? error;
    final requestKey = const Uuid().v4();
    if (!mounted) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (context) => StatefulBuilder(
        builder: (context, setSheet) => Padding(
          padding: EdgeInsets.fromLTRB(
            24,
            0,
            24,
            MediaQuery.viewInsetsOf(context).bottom + 24,
          ),
          child: SingleChildScrollView(
            child: Form(
              key: formKey,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    s.t('order'),
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 6),
                  Text(
                    '${value(product['name'])} · ₹ ${product['effective_price'] ?? product['price']} / L',
                  ),
                  const SizedBox(height: 22),
                  DropdownButtonFormField<String>(
                    initialValue: selected,
                    isExpanded: true,
                    decoration: InputDecoration(
                      labelText: s.t('address'),
                      prefixIcon: const Icon(Icons.location_on_outlined),
                    ),
                    items: [
                      for (final address in addresses)
                        DropdownMenuItem(
                          value: address['id'].toString(),
                          child: Text(
                            '${address['label']} · ${addressText(address)}',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                    ],
                    onChanged: saving
                        ? null
                        : (value) => setSheet(() => selected = value!),
                  ),
                  const SizedBox(height: 16),
                  field(s.t('date'), date),
                  field(s.t('quantity'), quantity, number: true),
                  field(s.t('note'), note, required: false),
                  if (error != null)
                    Container(
                      margin: const EdgeInsets.only(bottom: 16),
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: const Color(0xffFEF2F2),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        error!,
                        style: const TextStyle(color: Color(0xffB91C1C)),
                      ),
                    ),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton(
                          onPressed: saving
                              ? null
                              : () => Navigator.pop(context),
                          child: Text(s.t('Cancel')),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: FilledButton(
                          onPressed: saving
                              ? null
                              : () async {
                                  if (!formKey.currentState!.validate()) return;
                                  setSheet(() => saving = true);
                                  try {
                                    final address = addresses.firstWhere(
                                      (item) =>
                                          item['id'].toString() == selected,
                                    );
                                    final order = await vm.repository.mutate(
                                      'order_create',
                                      body: {
                                        'product_id': product['id'],
                                        'service_date': date.text.trim(),
                                        'quantity': quantity.text.trim(),
                                        'delivery_address': addressText(
                                          address,
                                        ),
                                        'customer_note': note.text.trim(),
                                      },
                                      key: requestKey,
                                    ) as Json;
                                    if (!context.mounted || !mounted) return;
                                    Navigator.pop(context);
                                    this.context.go(
                                      '/buyer/orders/${order['id']}',
                                    );
                                  } catch (e) {
                                    setSheet(() => error = e.toString());
                                  } finally {
                                    if (context.mounted) {
                                      setSheet(() => saving = false);
                                    }
                                  }
                                },
                          child: Text(s.t('order')),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
    date.dispose();
    quantity.dispose();
    note.dispose();
  }

  Future<void> result(Json row, bool delivered) async {
    if (delivered) {
      await form(
        'record_delivery',
        {
          'quantity': row['quantity']?.toString() ?? '',
          'unit_price': '',
          'note': '',
        },
        (body) => vm.repository.delivery(row['id'] as String, {
          ...body,
          'result': 'delivered',
        }),
      );
    } else {
      await form(
        s.t('not_delivered'),
        {'reason_code': 'other', 'note': ''},
        (body) => vm.repository.delivery(row['id'] as String, {
          ...body,
          'result': 'not_delivered',
        }),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    ref.watch(languageProvider);
    final sections = seller
        ? ['home', 'customers', 'orders', 'products', 'summary', 'profile']
        : ['home', 'orders', 'summary', 'profile'];
    final index = sections.indexOf(widget.section);
    return ListenableBuilder(
      listenable: vm,
      builder: (context, _) => Scaffold(
        appBar: AppBar(
          title: Text(
            widget.section == 'home' ? 'Doodhee' : s.t(widget.section),
          ),
          actions: [
            IconButton(
              tooltip: s.t('refresh'),
              onPressed: vm.loading ? null : load,
              icon: const Icon(Icons.refresh),
            ),
          ],
        ),
        bottomNavigationBar: NavigationBar(
          selectedIndex: index < 0 ? 0 : index,
          onDestinationSelected: (i) =>
              context.go('/${widget.role}/${sections[i]}'),
          destinations: [
            for (final section in sections)
              NavigationDestination(
                icon: Icon(switch (section) {
                  'home' => Icons.home_outlined,
                  'customers' => Icons.people_outline,
                  'orders' => Icons.receipt_long_outlined,
                  'products' => Icons.water_drop_outlined,
                  'summary' => Icons.bar_chart_outlined,
                  _ => Icons.person_outline,
                }),
                label: s.t(section),
              ),
          ],
        ),
        floatingActionButton:
            seller &&
                ['products', 'customers'].contains(widget.section) &&
                widget.id == null
            ? FloatingActionButton(
                onPressed: () => widget.section == 'products'
                    ? productForm()
                    : customerForm(),
                child: const Icon(Icons.add),
              )
            : null,
        body: RefreshIndicator(
          onRefresh: load,
          child: ListView(
            padding: const EdgeInsets.all(20),
            children: [
              if (ref.read(sessionProvider).offline)
                Card(
                  child: ListTile(
                    leading: const Icon(Icons.cloud_off_outlined),
                    title: Text(s.t('Offline · cached session')),
                    subtitle: Text(
                      s.t(
                        'Delivery changes will be verified when you reconnect.',
                      ),
                    ),
                  ),
                ),
              if (widget.section == 'home') ...[
                Text(
                  s.t('welcome'),
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 8),
                Text(
                  seller
                      ? s.t('Your daily deliveries, all in one place.')
                      : s.t('Discover milk sellers in your neighbourhood.'),
                ),
                const SizedBox(height: 24),
                if (seller)
                  Card(
                    child: ListTile(
                      leading: const Icon(Icons.add_shopping_cart_outlined),
                      title: Text(s.t('requests')),
                      subtitle: Text(
                        s.t(
                          'Approve or reject extra milk for the next delivery.',
                        ),
                      ),
                      trailing: const Icon(Icons.chevron_right),
                      onTap: () => context.go('/seller/requests'),
                    ),
                  ),
              ],
              if (seller && widget.section == 'customers' && widget.id == null)
                TextField(
                  decoration: InputDecoration(
                    labelText: s.t('search_customers'),
                    prefixIcon: const Icon(Icons.search),
                  ),
                  onSubmitted: (value) {
                    customerSearch = value.trim();
                    unawaited(load());
                  },
                ),
              if (vm.error != null)
                Card(
                  color: const Color(0xffFEF2F2),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      children: [
                        Text(
                          vm.stale
                              ? s.t('Offline · showing saved records')
                              : vm.error!,
                          style: const TextStyle(color: Color(0xffB91C1C)),
                        ),
                        TextButton(onPressed: load, child: Text(s.t('retry'))),
                      ],
                    ),
                  ),
                ),
              if (queued.isNotEmpty) ...[
                for (final q in queued)
                  Card(
                    child: ListTile(
                      leading: const Icon(Icons.sync),
                      title: Text(
                        s.t(q['state'] == 'conflict' ? 'attention' : 'sync'),
                      ),
                      subtitle: Text('${q['order_id']}\n${q['error'] ?? ''}'),
                    ),
                  ),
                TextButton(
                  onPressed: () async {
                    await vm.action(() => vm.repository.sync());
                    await load();
                  },
                  child: Text(s.t('refresh')),
                ),
              ],
              if (['summary', 'orders'].contains(widget.section) &&
                  widget.id == null)
                Row(
                  children: [
                    Expanded(
                      child: TextFormField(
                        key: ValueKey('from$from'),
                        initialValue: from,
                        decoration: InputDecoration(labelText: s.t('From')),
                        onFieldSubmitted: (v) {
                          from = v;
                          unawaited(load());
                        },
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: TextFormField(
                        key: ValueKey('to$to'),
                        initialValue: to,
                        decoration: InputDecoration(labelText: s.t('To')),
                        onFieldSubmitted: (v) {
                          to = v;
                          unawaited(load());
                        },
                      ),
                    ),
                  ],
                ),
              const SizedBox(height: 16),
              if (vm.loading && vm.data == null)
                ...List.generate(
                  3,
                  (_) => const Card(
                    child: SizedBox(
                      height: 100,
                      child: Center(child: CircularProgressIndicator()),
                    ),
                  ),
                )
              else if (widget.section == 'summary')
                summary(vm.data as Json? ?? {})
              else if (widget.section == 'profile')
                profile(vm.data as Json? ?? {})
              else if (widget.id != null && widget.section == 'orders')
                orderDetail(vm.data as Json? ?? {})
              else if (widget.id != null && widget.section == 'customers')
                customerDetail(vm.data as Json? ?? {})
              else if (widget.id != null && widget.section == 'sellers')
                sellerDetail(vm.data as Json? ?? {})
              else if (widget.id != null && widget.section == 'products')
                productCard(vm.data as Json? ?? {})
              else
                ...items(vm.data),
              if (vm.data is Map &&
                  vm.data['next_cursor'] != null &&
                  vm.data['next_cursor'] != '')
                TextButton(
                  onPressed: vm.busy ? null : vm.more,
                  child: Text(s.t('Load more')),
                ),
            ],
          ),
        ),
      ),
    );
  }

  List<Widget> items(dynamic data) {
    final rows = (data is Map ? data['items'] : null) as List? ?? [];
    if (rows.isEmpty) {
      return [
        Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            children: [
              const Icon(
                Icons.local_drink_outlined,
                size: 48,
                color: Color(0xff166534),
              ),
              const SizedBox(height: 16),
              Text(s.t('empty')),
            ],
          ),
        ),
      ];
    }
    return rows.map((e) {
      final r = Map<String, dynamic>.from(e as Map);
      if (widget.section == 'customers') {
        return Card(
          child: ListTile(
            contentPadding: const EdgeInsets.all(16),
            leading: const CircleAvatar(
              backgroundColor: Color(0xffDCFCE7),
              child: Icon(Icons.person_outline, color: Color(0xff166534)),
            ),
            title: Text(value(r['full_name'])),
            subtitle: Text('${r['phone']}\n${r['delivery_address']}'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.go('/seller/customers/${r['id']}'),
          ),
        );
      }
      if (widget.section == 'requests') {
        return Card(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      '${r['extra_quantity']} L extra',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    StatusChip(value(r['status']), s.t(value(r['status']))),
                  ],
                ),
                const SizedBox(height: 8),
                Text(
                  '${value(r['full_name'])} · ${value(r['product_name'])}\n'
                  '${value(r['service_date'])} · planned ${value(r['planned_quantity'])} L',
                ),
                if (r['status'] == 'pending') ...[
                  const SizedBox(height: 14),
                  Row(
                    children: [
                      Expanded(
                        child: FilledButton(
                          onPressed: vm.busy
                              ? null
                              : () => mutate(
                                  'seller_extra_approve',
                                  id: r['id'] as String,
                                  body: {'decision_note': ''},
                                ),
                          child: Text(s.t('approve')),
                        ),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: OutlinedButton(
                          onPressed: vm.busy
                              ? null
                              : () => mutate(
                                  'seller_extra_reject',
                                  id: r['id'] as String,
                                  body: {'decision_note': ''},
                                ),
                          child: Text(s.t('reject')),
                        ),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
        );
      }
      if (widget.section == 'products' || widget.section == 'sellers') {
        return productCard(r);
      }
      if (widget.section == 'orders' || seller && widget.section == 'home') {
        return Card(
          child: ListTile(
            contentPadding: const EdgeInsets.all(16),
            leading: const Icon(
              Icons.local_shipping_outlined,
              color: Color(0xff166534),
            ),
            title: Text('${r['service_date']} · ${r['quantity']} L'),
            subtitle: Text(
              '₹ ${r['total_amount']}\n${r['not_delivered_reason'] ?? ''}',
            ),
            trailing: StatusChip(value(r['status']), s.t(value(r['status']))),
            onTap: () => context.go('/${widget.role}/orders/${r['id']}'),
          ),
        );
      }
      return Card(
        child: ListTile(
          contentPadding: const EdgeInsets.all(18),
          leading: const CircleAvatar(
            backgroundColor: Color(0xffDCFCE7),
            child: Icon(Icons.storefront_outlined, color: Color(0xff166534)),
          ),
          title: Text(value(r['full_name'])),
          subtitle: Text(
            '${r['city']} · ${r['pincode']}\n#${r['rank'] ?? '—'} · ⭐ ${r['average_rating'] ?? '0.00'} (${r['rating_count'] ?? 0})',
          ),
          trailing: const Icon(Icons.chevron_right),
          onTap: () => context.go('/buyer/sellers/${r['id']}'),
        ),
      );
    }).toList();
  }

  Widget productCard(Json r) => Card(
    child: Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(
            Icons.water_drop_outlined,
            color: Color(0xff166534),
            size: 32,
          ),
          const SizedBox(height: 12),
          Text(value(r['name']), style: Theme.of(context).textTheme.titleLarge),
          Text('₹ ${r['effective_price'] ?? r['price']} / litre'),
          if (!seller && r['price_source'] == 'customer')
            Text(
              s.t('customer_price'),
              style: const TextStyle(color: Color(0xff166534)),
            ),
          const SizedBox(height: 8),
          if (seller) ...[
            Text('${r['daily_capacity']} L daily capacity'),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: Text(s.t('Available for orders')),
              value: r['is_available'] == true,
              onChanged: vm.busy
                  ? null
                  : (v) => mutate(
                      'seller_product_availability',
                      id: r['id'] as String,
                      body: {'is_available': v},
                    ),
            ),
            OutlinedButton(
              onPressed: () => productForm(r),
              child: Text(s.t('Edit product')),
            ),
          ] else
            FilledButton(
              onPressed: r.isEmpty ? null : () => orderForm(r),
              child: Text(s.t('order')),
            ),
        ],
      ),
    ),
  );
  Widget customerDetail(Json customer) {
    final products = (customer['products'] as List? ?? [])
        .map((item) => Map<String, dynamic>.from(item as Map))
        .toList();
    final prices = (customer['prices'] as List? ?? [])
        .map((item) => Map<String, dynamic>.from(item as Map))
        .toList();
    for (final product in products) {
      for (final price in prices) {
        if (price['product_id'] == product['id']) {
          product['customer_price'] = price['price'];
        }
      }
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Card(
          child: ListTile(
            contentPadding: const EdgeInsets.all(20),
            leading: const CircleAvatar(
              backgroundColor: Color(0xffDCFCE7),
              child: Icon(Icons.person_outline, color: Color(0xff166534)),
            ),
            title: Text(
              value(customer['full_name']),
              style: Theme.of(context).textTheme.titleLarge,
            ),
            subtitle: Text(
              '${customer['phone']}\n${customer['delivery_address']}\n${customer['phone_verified_at'] == null ? 'Waiting for first OTP login' : 'Phone verified'}',
            ),
          ),
        ),
        const SizedBox(height: 8),
        for (final product in products)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    value(product['name']),
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  Text(
                    product['customer_price'] == null
                        ? 'Default ₹ ${product['price']} / L'
                        : 'Customer ₹ ${product['customer_price']} / L · default ₹ ${product['price']}',
                  ),
                  const SizedBox(height: 14),
                  FilledButton.icon(
                    onPressed: vm.busy
                        ? null
                        : () => directDeliveryForm(customer, product),
                    icon: const Icon(Icons.check_circle_outline),
                    label: Text(s.t('record_delivery')),
                  ),
                  const SizedBox(height: 8),
                  OutlinedButton(
                    onPressed: vm.busy
                        ? null
                        : () => customerPriceForm(customer, product),
                    child: Text(s.t('customer_price')),
                  ),
                  if (product['customer_price'] != null)
                    TextButton(
                      onPressed: vm.busy
                          ? null
                          : () => mutate(
                              'seller_customer_price_delete',
                              id: customer['id'] as String,
                              body: {'product_id': product['id']},
                            ),
                      child: Text(s.t('use_default_price')),
                    ),
                ],
              ),
            ),
          ),
      ],
    );
  }

  Widget sellerDetail(Json data) {
    final milkman = Map<String, dynamic>.from(
      (data['seller'] as Map?) ?? <String, dynamic>{},
    );
    final rating = Map<String, dynamic>.from(
      (data['rating'] as Map?) ?? <String, dynamic>{},
    );
    final products = (data['items'] as List? ?? [])
        .map((item) => Map<String, dynamic>.from(item as Map))
        .toList();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Card(
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  value(milkman['full_name']),
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                Text(
                  '#${milkman['rank'] ?? '—'} · ⭐ ${milkman['average_rating'] ?? '0.00'} (${milkman['rating_count'] ?? 0})',
                ),
                const SizedBox(height: 16),
                Text(s.t('rating')),
                Wrap(
                  spacing: 4,
                  children: [
                    for (var value = 1; value <= 5; value++)
                      IconButton(
                        tooltip: '$value / 5',
                        onPressed: vm.busy
                            ? null
                            : () => mutate(
                                'rating_set',
                                id: widget.id,
                                body: {'rating': '$value'},
                              ),
                        icon: Icon(
                          value <= (rating['rating'] as int? ?? 0)
                              ? Icons.star
                              : Icons.star_border,
                          color: const Color(0xffD97706),
                        ),
                      ),
                  ],
                ),
              ],
            ),
          ),
        ),
        for (final product in products) productCard(product),
      ],
    );
  }

  Widget orderDetail(Json r) => Card(
    child: Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          StatusChip(value(r['status']), s.t(value(r['status']))),
          const SizedBox(height: 16),
          Text(
            '${r['service_date']}',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          Text('${r['quantity']} L × ₹ ${r['unit_price']}'),
          if (r['base_quantity'] != null)
            Text(
              'Base ${r['base_quantity']} L · approved extra ${r['approved_extra_quantity'] ?? '0.000'} L',
            ),
          Text('${s.t('price_source')}: ${r['price_source'] ?? 'default'}'),
          Text(
            '₹ ${r['total_amount']}',
            style: Theme.of(context).textTheme.headlineMedium,
          ),
          const Divider(height: 32),
          Text(value(r['delivery_address'])),
          Text(value(r['customer_note'])),
          if (r['not_delivered_reason'] != null)
            Text('${s.t('reason')}: ${r['not_delivered_reason']}'),
          if (r['delivery_note'] != '') Text(value(r['delivery_note'])),
          if (r['extra_request'] is Map &&
              r['extra_request']['status'] != 'none')
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: ListTile(
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.add_shopping_cart_outlined),
                title: Text(
                  '${r['extra_request']['extra_quantity']} L extra milk',
                ),
                trailing: StatusChip(
                  value(r['extra_request']['status']),
                  s.t(value(r['extra_request']['status'])),
                ),
              ),
            ),
          const SizedBox(height: 20),
          if (seller && r['status'] == 'pending')
            FilledButton(
              onPressed: vm.busy
                  ? null
                  : () => mutate('seller_order_accept', id: r['id'] as String),
              child: Text(s.t('Accept order')),
            ),
          if (seller && r['status'] == 'accepted')
            FilledButton(
              onPressed: vm.busy
                  ? null
                  : () =>
                        mutate('seller_order_dispatch', id: r['id'] as String),
              child: Text(s.t('out_for_delivery')),
            ),
          if (seller && r['status'] == 'out_for_delivery') ...[
            FilledButton(
              onPressed: vm.busy ? null : () => result(r, true),
              child: Text(s.t('delivered')),
            ),
            const SizedBox(height: 12),
            OutlinedButton(
              onPressed: vm.busy ? null : () => result(r, false),
              child: Text(s.t('not_delivered')),
            ),
          ],
          if (!seller &&
              ['pending', 'accepted'].contains(r['status']) &&
              (r['extra_request'] is! Map ||
                  ['none', 'rejected'].contains(r['extra_request']['status'])))
            FilledButton.icon(
              onPressed: vm.busy ? null : () => extraMilkForm(r),
              icon: const Icon(Icons.add),
              label: Text(s.t('extra_milk')),
            ),
          if (!seller && ['pending', 'accepted'].contains(r['status'])) ...[
            const SizedBox(height: 10),
            OutlinedButton(
              onPressed: vm.busy
                  ? null
                  : () => mutate(
                      'order_cancel',
                      id: r['id'] as String,
                      body: {'reason': 'Buyer cancelled before delivery'},
                    ),
              child: Text(s.t('Cancel order')),
            ),
          ],
        ],
      ),
    ),
  );
  Widget summary(Json r) => Column(
    children: [
      for (final pair in [
        ['days', 'delivered_days'],
        ['litres', 'delivered_litres'],
        ['total', 'total_amount'],
      ])
        Card(
          child: ListTile(
            contentPadding: const EdgeInsets.all(20),
            title: Text(s.t(pair[0])),
            trailing: Text(
              '${pair[0] == 'total' ? '₹ ' : ''}${r[pair[1]] ?? '0'}',
              style: Theme.of(context).textTheme.titleLarge,
            ),
          ),
        ),
      for (final group in ((r['groups'] as Map?) ?? {}).values)
        Card(
          child: ListTile(
            title: Text(value(group['full_name'] ?? group['user_id'])),
            subtitle: Text('${group['delivered_litres']} L'),
            trailing: Text('₹ ${group['total_amount']}'),
          ),
        ),
    ],
  );
  Future<void> addressForm([Json? address]) => form(
    address == null ? 'add_address' : 'edit_address',
    {
      'label': address?['label']?.toString() ?? '',
      'address_line': address?['address_line']?.toString() ?? '',
      'city': address?['city']?.toString() ?? '',
      'pincode': address?['pincode']?.toString() ?? '',
    },
    (body) async {
      await vm.repository.mutate(
        address == null ? 'address_create' : 'address_update',
        id: address?['id'] as String?,
        body: body,
      );
    },
  );

  Future<bool> confirmSheet(
    String title,
    String message,
    String action,
  ) async =>
      await showModalBottomSheet<bool>(
        context: context,
        useSafeArea: true,
        showDragHandle: true,
        builder: (context) => Padding(
          padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(title, style: Theme.of(context).textTheme.headlineSmall),
              const SizedBox(height: 8),
              Text(message),
              const SizedBox(height: 24),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () => Navigator.pop(context, false),
                      child: Text(s.t('Cancel')),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      onPressed: () => Navigator.pop(context, true),
                      child: Text(action),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ) ??
      false;

  Future<void> removeAddress(Json address) async {
    if (!await confirmSheet(
      s.t('remove_address_title'),
      s.t('remove_address_help'),
      s.t('remove'),
    )) {
      return;
    }
    await mutate('address_delete', id: address['id'] as String);
  }

  Future<void> logout() async {
    if (!await confirmSheet(
      s.t('logout_title'),
      s.t('otp_security_help'),
      s.t('logout'),
    )) {
      return;
    }
    await vm.action(() async {
      await ref.read(sessionProvider).logout();
      await ref.read(localProvider).clear();
    });
  }

  Widget profile(Json r) {
    final addresses = (r['addresses'] as List? ?? [])
        .map((item) => Map<String, dynamic>.from(item as Map))
        .toList();
    final name = value(r['full_name']);
    final initial = name.isEmpty || name == '—' ? 'D' : name[0].toUpperCase();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Container(
          padding: const EdgeInsets.all(22),
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              colors: [Color(0xff14532D), Color(0xff15803D)],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            borderRadius: BorderRadius.circular(22),
          ),
          child: Row(
            children: [
              CircleAvatar(
                radius: 31,
                backgroundColor: Colors.white,
                child: Text(
                  initial,
                  style: const TextStyle(
                    color: Color(0xff166534),
                    fontSize: 25,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      name,
                      style: Theme.of(context).textTheme.titleLarge?.copyWith(
                        color: Colors.white,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      s.t(seller ? 'seller' : 'buyer'),
                      style: const TextStyle(color: Color(0xffDCFCE7)),
                    ),
                  ],
                ),
              ),
              const Icon(Icons.verified_rounded, color: Color(0xffBBF7D0)),
            ],
          ),
        ),
        const SizedBox(height: 18),
        Card(
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        s.t('personal_details'),
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                    ),
                    TextButton.icon(
                      onPressed: () => form(
                        'Edit profile',
                        seller
                            ? {
                                'full_name': name,
                                'address_line':
                                    r['address_line']?.toString() ?? '',
                                'city': r['city']?.toString() ?? '',
                                'pincode': r['pincode']?.toString() ?? '',
                              }
                            : {'full_name': name},
                        (body) async {
                          await vm.repository.mutate(
                            seller ? 'seller_profile_update' : 'profile',
                            body: body,
                          );
                          await ref.read(sessionProvider).reload();
                        },
                      ),
                      icon: const Icon(Icons.edit_outlined),
                      label: Text(s.t('edit_address').split(' ').first),
                    ),
                  ],
                ),
                const Divider(height: 28),
                ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: const CircleAvatar(
                    backgroundColor: Color(0xffECFDF3),
                    child: Icon(Icons.phone_outlined, color: Color(0xff166534)),
                  ),
                  title: Text(s.t('verified_phone')),
                  subtitle: Text(value(r['phone'])),
                  trailing: const Icon(
                    Icons.check_circle,
                    color: Color(0xff15803D),
                  ),
                ),
                ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: const CircleAvatar(
                    backgroundColor: Color(0xffECFDF3),
                    child: Icon(
                      Icons.shield_outlined,
                      color: Color(0xff166534),
                    ),
                  ),
                  title: Text(s.t('account_status')),
                  subtitle: Text(
                    s.t(
                      r['is_enabled'] == true ? 'active_account' : 'disabled',
                    ),
                  ),
                ),
                if (seller) ...[
                  const Divider(height: 28),
                  Text(
                    s.t('seller_business'),
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    addressText(r),
                    style: Theme.of(context).textTheme.bodyLarge,
                  ),
                ],
              ],
            ),
          ),
        ),
        if (!seller) ...[
          const SizedBox(height: 10),
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      s.t('saved_addresses'),
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    Text(s.t('addresses_help')),
                  ],
                ),
              ),
              FilledButton.tonalIcon(
                onPressed: vm.busy ? null : () => addressForm(),
                icon: const Icon(Icons.add_location_alt_outlined),
                label: Text(s.t('add_address')),
              ),
            ],
          ),
          const SizedBox(height: 10),
          if (addresses.isEmpty)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  children: [
                    const Icon(
                      Icons.location_off_outlined,
                      size: 38,
                      color: Color(0xff64748B),
                    ),
                    const SizedBox(height: 10),
                    Text(s.t('no_addresses')),
                    const SizedBox(height: 12),
                    OutlinedButton.icon(
                      onPressed: () => addressForm(),
                      icon: const Icon(Icons.add),
                      label: Text(s.t('add_address')),
                    ),
                  ],
                ),
              ),
            )
          else
            for (final address in addresses)
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(18),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Row(
                        children: [
                          const CircleAvatar(
                            backgroundColor: Color(0xffECFDF3),
                            child: Icon(
                              Icons.location_on_outlined,
                              color: Color(0xff166534),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Text(
                              value(address['label']),
                              style: Theme.of(context).textTheme.titleMedium,
                            ),
                          ),
                          if (address['is_default'] == true)
                            Chip(
                              label: Text(s.t('default')),
                              avatar: const Icon(Icons.check, size: 16),
                              side: BorderSide.none,
                              backgroundColor: const Color(0xffDCFCE7),
                            ),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Text(addressText(address)),
                      const SizedBox(height: 12),
                      Wrap(
                        spacing: 8,
                        runSpacing: 6,
                        children: [
                          TextButton.icon(
                            onPressed: vm.busy
                                ? null
                                : () => addressForm(address),
                            icon: const Icon(Icons.edit_outlined),
                            label: Text(s.t('edit_address')),
                          ),
                          if (address['is_default'] != true)
                            TextButton.icon(
                              onPressed: vm.busy
                                  ? null
                                  : () => mutate(
                                      'address_default',
                                      id: address['id'] as String,
                                    ),
                              icon: const Icon(Icons.star_outline),
                              label: Text(s.t('set_default')),
                            ),
                          TextButton.icon(
                            onPressed: vm.busy
                                ? null
                                : () => removeAddress(address),
                            icon: const Icon(Icons.delete_outline),
                            label: Text(s.t('remove')),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
        ],
        const SizedBox(height: 10),
        Card(
          child: ListTile(
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 20,
              vertical: 8,
            ),
            leading: const CircleAvatar(
              backgroundColor: Color(0xffECFDF3),
              child: Icon(Icons.language, color: Color(0xff166534)),
            ),
            title: Text(s.t('language')),
            subtitle: Text(s.t('preferences')),
            trailing: languagePicker(ref),
          ),
        ),
        const SizedBox(height: 14),
        OutlinedButton.icon(
          onPressed: vm.busy ? null : logout,
          icon: const Icon(Icons.logout),
          label: Text(s.t('logout')),
          style: OutlinedButton.styleFrom(
            foregroundColor: const Color(0xffB91C1C),
          ),
        ),
        const SizedBox(height: 24),
      ],
    );
  }
}
