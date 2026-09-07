import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_frontend/order_history.dart';
import 'helpers.dart';

void main() {
  test('delivery states distinguish confirmation, retry and DLQ', () {
    expect(deliveryLabel(null), 'Sin información');
    expect(deliveryLabel({'done': false, 'failures': 0}), 'Pendiente');
    expect(deliveryLabel({'done': true}), 'Confirmado');
    expect(deliveryLabel({'done': false, 'failures': 2}), 'Reintentando');
    expect(deliveryLabel({'done': false, 'dead': true}), 'Falló · en DLQ');
  });
  testWidgets(
    'history opens full order details and independent destination states',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final api = FakeApi();
      final order = sampleOrder(api.result);
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: OrderHistoryCard(order: order, api: api),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('ERP: Confirmado'), findsOneWidget);
      expect(find.text('Notificaciones PUSH: Reintentando'), findsOneWidget);
      await tester.tap(find.text('Ver detalle del pedido ${order.id}'));
      await tester.pumpAndSettle();
      expect(find.text('Detalle del pedido'), findsOneWidget);
      expect(find.text('Productos confirmados'), findsOneWidget);
      expect(find.text('Total confirmado'), findsOneWidget);
      expect(find.text(historyDemoNotice), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.tap(find.byTooltip('Cerrar detalle'));
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox());
    },
  );
}
