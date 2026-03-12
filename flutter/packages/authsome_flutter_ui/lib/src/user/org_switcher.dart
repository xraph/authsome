/// Organization switcher widget — lists and switches between organizations.
///
/// Fetches the user's organizations on init and displays them in a
/// [PopupMenuButton]. The active organization is indicated with a
/// checkmark. Includes a "Create organization" option that opens a
/// dialog for creating a new organization with name and slug.
///
/// ```dart
/// OrgSwitcher(
///   activeOrgId: currentOrgId,
///   onOrgChange: (orgId) => setState(() => currentOrgId = orgId),
///   onCreateOrg: () => print('Org created'),
/// )
/// ```
library;

import 'package:flutter/material.dart';
import 'package:authsome_flutter/authsome_flutter.dart';

/// A popup menu for switching between organizations and creating new ones.
///
/// On mount, fetches the organization list from
/// `auth.client.listOrganizations`. Each organization is shown as a
/// menu item; the one matching [activeOrgId] displays a leading
/// checkmark. A footer option opens a dialog to create a new
/// organization.
class OrgSwitcher extends StatefulWidget {
  /// Called when the user selects a different organization.
  final ValueChanged<String>? onOrgChange;

  /// Called after a new organization is created.
  final VoidCallback? onCreateOrg;

  /// The ID of the currently active organization.
  final String? activeOrgId;

  /// Creates an [OrgSwitcher].
  const OrgSwitcher({
    this.onOrgChange,
    this.onCreateOrg,
    this.activeOrgId,
    super.key,
  });

  @override
  State<OrgSwitcher> createState() => _OrgSwitcherState();
}

class _OrgSwitcherState extends State<OrgSwitcher> {
  List<Organization> _orgs = [];
  bool _loading = true;
  String? _error;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _fetchOrgs();
  }

  Future<void> _fetchOrgs() async {
    final auth = context.auth;
    final token = auth.session?.sessionToken ?? '';

    try {
      final response = await auth.client.listOrganizations(token: token);
      if (!mounted) return;

      setState(() {
        _orgs = response.organizations;
        _loading = false;
        _error = null;
      });
    } catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = e.toString();
        });
      }
    }
  }

  /// Reads a field from an Organization object.
  String _orgField(Organization org, String key) {
    return switch (key) {
      'id' => org.id,
      'name' => org.name,
      'slug' => org.slug,
      _ => '',
    };
  }

  /// Generates a URL-safe slug from a name.
  String _slugify(String name) {
    return name
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9\s-]'), '')
        .replaceAll(RegExp(r'\s+'), '-')
        .replaceAll(RegExp(r'-+'), '-')
        .replaceAll(RegExp(r'^-|-$'), '');
  }

  Future<void> _showCreateDialog() async {
    final nameController = TextEditingController();
    final slugController = TextEditingController();
    bool autoSlug = true;

    final created = await showDialog<bool>(
      context: context,
      builder: (dialogContext) {
        return StatefulBuilder(
          builder: (dialogContext, setDialogState) {
            return AlertDialog(
              title: const Text('Create Organization'),
              content: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: nameController,
                    decoration: InputDecoration(
                      labelText: 'Name',
                      hintText: 'My Organization',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    onChanged: (value) {
                      if (autoSlug) {
                        slugController.text = _slugify(value);
                      }
                    },
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: slugController,
                    decoration: InputDecoration(
                      labelText: 'Slug',
                      hintText: 'my-organization',
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    onChanged: (_) {
                      autoSlug = false;
                    },
                  ),
                ],
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(dialogContext).pop(false),
                  child: const Text('Cancel'),
                ),
                FilledButton(
                  onPressed: () async {
                    final name = nameController.text.trim();
                    final slug = slugController.text.trim();
                    if (name.isEmpty) return;

                    try {
                      final auth = context.auth;
                      final token = auth.session?.sessionToken ?? '';
                      await auth.client.createOrganization(
                        body: {
                          'name': name,
                          'slug': slug.isNotEmpty ? slug : _slugify(name),
                        },
                        token: token,
                      );
                      if (dialogContext.mounted) {
                        Navigator.of(dialogContext).pop(true);
                      }
                    } catch (e) {
                      if (dialogContext.mounted) {
                        ScaffoldMessenger.of(dialogContext).showSnackBar(
                          SnackBar(
                            content: Text('Failed to create organization: $e'),
                            backgroundColor:
                                Theme.of(dialogContext).colorScheme.error,
                          ),
                        );
                      }
                    }
                  },
                  child: const Text('Create'),
                ),
              ],
            );
          },
        );
      },
    );

    // Clean up controllers.
    nameController.dispose();
    slugController.dispose();

    if (created == true) {
      widget.onCreateOrg?.call();
      _fetchOrgs();
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;

    if (_loading) {
      return const SizedBox(
        width: 24,
        height: 24,
        child: CircularProgressIndicator(strokeWidth: 2),
      );
    }

    if (_error != null) {
      return Tooltip(
        message: 'Failed to load organizations',
        child: IconButton(
          icon: Icon(Icons.error_outline, color: colorScheme.error),
          onPressed: _fetchOrgs,
        ),
      );
    }

    // Determine the label for the button.
    String activeLabel = 'Select organization';
    for (final org in _orgs) {
      if (_orgField(org, 'id') == widget.activeOrgId) {
        activeLabel = _orgField(org, 'name');
        break;
      }
    }

    return PopupMenuButton<String>(
      offset: const Offset(0, 40),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      itemBuilder: (context) {
        final items = <PopupMenuEntry<String>>[];

        for (final org in _orgs) {
          final id = _orgField(org, 'id');
          final name = _orgField(org, 'name');
          final isActive = id == widget.activeOrgId;

          items.add(
            PopupMenuItem<String>(
              value: id,
              child: Row(
                children: [
                  if (isActive)
                    Icon(Icons.check, size: 18, color: colorScheme.primary)
                  else
                    const SizedBox(width: 18),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      name,
                      style: isActive
                          ? textTheme.bodyMedium?.copyWith(
                              fontWeight: FontWeight.w600,
                            )
                          : textTheme.bodyMedium,
                    ),
                  ),
                ],
              ),
            ),
          );
        }

        items.add(const PopupMenuDivider());

        items.add(
          const PopupMenuItem<String>(
            value: '_create',
            child: Row(
              children: [
                Icon(Icons.add, size: 18),
                SizedBox(width: 8),
                Text('Create organization'),
              ],
            ),
          ),
        );

        return items;
      },
      onSelected: (value) {
        if (value == '_create') {
          _showCreateDialog();
        } else {
          widget.onOrgChange?.call(value);
        }
      },
      child: Chip(
        avatar: Icon(Icons.business, size: 18, color: colorScheme.onSurface),
        label: Text(activeLabel),
      ),
    );
  }
}
