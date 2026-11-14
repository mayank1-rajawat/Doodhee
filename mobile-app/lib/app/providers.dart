import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/api.dart';
import '../core/repository.dart';

final apiProvider = Provider((ref) => ApiService());
final sessionProvider = Provider(
  (ref) => SessionRepository(ref.watch(apiProvider)),
);
final localProvider = Provider((ref) => LocalStore());
final repositoryProvider = Provider(
  (ref) => MarketplaceRepository(
    ref.watch(apiProvider),
    ref.watch(localProvider),
    ref.watch(sessionProvider),
  ),
);

class Language extends Notifier<String> {
  @override
  String build() => 'en';
  void change(String value) => state = value;
}

final languageProvider = NotifierProvider<Language, String>(Language.new);
