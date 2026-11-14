import 'package:material_ui/material_ui.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../app/providers.dart';
import '../app/strings.dart';

Future<void> editPendingProfile(BuildContext context, WidgetRef ref) async {
  final session = ref.read(sessionProvider);
  final s = Strings(ref.read(languageProvider));
  final controllers = {
    for (final k in ['full_name', 'address_line', 'city', 'pincode'])
      k: TextEditingController(text: session.user?[k]?.toString() ?? ''),
  };
  var busy = false;
  String? error;
  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    showDragHandle: true,
    builder: (context) => StatefulBuilder(
      builder: (context, setState) => Padding(
        padding: EdgeInsets.fromLTRB(
          24,
          0,
          24,
          MediaQuery.viewInsetsOf(context).bottom + 24,
        ),
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                s.t('profile'),
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              const SizedBox(height: 8),
              Text(s.t('Complete the details below.')),
              const SizedBox(height: 22),
              for (final e in controllers.entries)
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: TextField(
                    controller: e.value,
                    decoration: InputDecoration(
                      labelText: s.t(e.key.replaceAll('_', ' ')),
                    ),
                  ),
                ),
              if (error != null)
                Container(
                  padding: const EdgeInsets.all(12),
                  margin: const EdgeInsets.only(bottom: 16),
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
                      onPressed: busy ? null : () => Navigator.pop(context),
                      child: Text(s.t('Cancel')),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      onPressed: busy
                          ? null
                          : () async {
                              setState(() => busy = true);
                              try {
                                await ref
                                    .read(repositoryProvider)
                                    .mutate(
                                      'seller_profile_update',
                                      body: controllers.map(
                                        (k, v) => MapEntry(k, v.text.trim()),
                                      ),
                                    );
                                await session.reload();
                                if (context.mounted) Navigator.pop(context);
                              } catch (e) {
                                setState(() => error = e.toString());
                              } finally {
                                if (context.mounted) {
                                  setState(() => busy = false);
                                }
                              }
                            },
                      child: Text(s.t('save')),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    ),
  );
  for (final c in controllers.values) {
    c.dispose();
  }
}
