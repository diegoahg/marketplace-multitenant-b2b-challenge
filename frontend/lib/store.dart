import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import 'models.dart';

abstract class CheckoutStorage {
  Future<Json?> read();
  Future<void> write(Json state);
}

class LocalCheckoutStorage implements CheckoutStorage {
  static const key = 'abasto.checkout.v1';
  @override
  Future<Json?> read() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(key);
    return raw == null ? null : jsonDecode(raw) as Json;
  }

  @override
  Future<void> write(Json state) async {
    final prefs = await SharedPreferences.getInstance();
    if (!await prefs.setString(key, jsonEncode(state))) {
      throw StateError('Cannot persist checkout');
    }
  }
}
