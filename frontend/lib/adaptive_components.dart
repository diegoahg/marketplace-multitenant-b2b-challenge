import 'package:flutter/material.dart';

/// One owner for system insets and keyboard resizing. Forms remain scrollable.
class SafeAppScaffold extends StatelessWidget {
  final Widget body;
  final Widget? navigation;
  const SafeAppScaffold({super.key, required this.body, this.navigation});
  @override
  Widget build(BuildContext context) {
    final keyboardOpen = MediaQuery.viewInsetsOf(context).bottom > 0;
    final bar = keyboardOpen ? null : navigation;
    return Scaffold(
      resizeToAvoidBottomInset: true,
      body: SafeArea(bottom: bar == null, child: body),
      bottomNavigationBar: bar == null
          ? null
          : SafeArea(top: false, child: bar),
    );
  }
}

/// Text keeps its natural height. Narrow or enlarged layouts stack both sides.
class AdaptivePair extends StatelessWidget {
  final Widget primary, secondary;
  final double breakpoint;
  const AdaptivePair({
    super.key,
    required this.primary,
    required this.secondary,
    this.breakpoint = 360,
  });
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final enlarged = MediaQuery.textScalerOf(context).scale(14) / 14;
      if (constraints.maxWidth < breakpoint * enlarged) {
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            primary,
            const SizedBox(height: 8),
            Align(alignment: Alignment.centerRight, child: secondary),
          ],
        );
      }
      return Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(child: primary),
          const SizedBox(width: 12),
          Flexible(
            flex: 2,
            child: Align(alignment: Alignment.centerRight, child: secondary),
          ),
        ],
      );
    },
  );
}

/// Every tile shares the same viewport, padding and contain fit (never crop).
class ProductMedia extends StatelessWidget {
  static const ratio = 4 / 3;
  static const inset = 16.0;
  final Widget child;
  const ProductMedia({super.key, required this.child});
  @override
  Widget build(BuildContext context) => AspectRatio(
    aspectRatio: ratio,
    child: Padding(
      padding: const EdgeInsets.all(inset),
      child: FittedBox(
        fit: BoxFit.contain,
        alignment: Alignment.center,
        child: child,
      ),
    ),
  );
}

/// Grid height is content-driven; larger text reduces the number of columns.
class ProductGrid extends StatelessWidget {
  final List<Widget> children;
  const ProductGrid({super.key, required this.children});
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, box) {
      final scale = MediaQuery.textScalerOf(context).scale(14) / 14;
      final twoColumns = box.maxWidth >= 560 * scale;
      final width = twoColumns ? (box.maxWidth - 16) / 2 : box.maxWidth;
      return Wrap(
        spacing: 16,
        runSpacing: 16,
        children: children
            .map((child) => SizedBox(width: width, child: child))
            .toList(),
      );
    },
  );
}

/// Dialog content scrolls above persistent actions when the keyboard opens.
class SafeFormDialog extends StatelessWidget {
  final Widget title, content;
  final List<Widget> actions;
  const SafeFormDialog({
    super.key,
    required this.title,
    required this.content,
    required this.actions,
  });
  @override
  Widget build(BuildContext context) => AlertDialog(
    scrollable: true,
    insetPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
    title: title,
    content: content,
    actions: actions,
  );
}
