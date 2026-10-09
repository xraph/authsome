library;

bool validEmail(String value) => RegExp(r'^[^\s@]+@[^\s@]+$').hasMatch(value);
