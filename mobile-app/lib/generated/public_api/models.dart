// Generated. Run make contracts.
class AddressDto {
  final Map<String, dynamic> json;
  const AddressDto(this.json);
  factory AddressDto.fromJson(Map<String, dynamic> json) => AddressDto(json);
  Map<String, dynamic> toJson() => json;
  String? get addressLine => json['address_line'] as String?;
  String? get city => json['city'] as String?;
  String? get createdAt => json['created_at'] as String?;
  String? get id => json['id'] as String?;
  bool? get isDefault => json['is_default'] as bool?;
  String? get label => json['label'] as String?;
  String? get pincode => json['pincode'] as String?;
  String? get updatedAt => json['updated_at'] as String?;
  String? get userId => json['user_id'] as String?;
}

class AuditDto {
  final Map<String, dynamic> json;
  const AuditDto(this.json);
  factory AuditDto.fromJson(Map<String, dynamic> json) => AuditDto(json);
  Map<String, dynamic> toJson() => json;
  String? get action => json['action'] as String?;
  String? get adminId => json['admin_id'] as String?;
  dynamic get afterData => json['after_data'];
  dynamic get beforeData => json['before_data'];
  String? get createdAt => json['created_at'] as String?;
  String? get id => json['id'] as String?;
  String? get reason => json['reason'] as String?;
  String? get requestId => json['request_id'] as String?;
  String? get targetId => json['target_id'] as String?;
  String? get targetType => json['target_type'] as String?;
}

class ChallengeDto {
  final Map<String, dynamic> json;
  const ChallengeDto(this.json);
  factory ChallengeDto.fromJson(Map<String, dynamic> json) =>
      ChallengeDto(json);
  Map<String, dynamic> toJson() => json;
  String? get challengeId => json['challenge_id'] as String?;
  int? get expiresIn => json['expires_in'] as int?;
  String? get testingOtp => json['testing_otp'] as String?;
}

class CustomerLookupDto {
  final Map<String, dynamic> json;
  const CustomerLookupDto(this.json);
  factory CustomerLookupDto.fromJson(Map<String, dynamic> json) =>
      CustomerLookupDto(json);
  Map<String, dynamic> toJson() => json;
  bool? get alreadyConnected => json['already_connected'] as bool?;
  String? get deliveryAddress => json['delivery_address'] as String?;
  bool? get exists => json['exists'] as bool?;
  String? get fullName => json['full_name'] as String?;
  String? get id => json['id'] as String?;
  String? get phone => json['phone'] as String?;
  String? get phoneVerifiedAt => json['phone_verified_at'] as String?;
}

class CustomerPriceDto {
  final Map<String, dynamic> json;
  const CustomerPriceDto(this.json);
  factory CustomerPriceDto.fromJson(Map<String, dynamic> json) =>
      CustomerPriceDto(json);
  Map<String, dynamic> toJson() => json;
  String? get createdAt => json['created_at'] as String?;
  String? get customerId => json['customer_id'] as String?;
  String? get id => json['id'] as String?;
  String? get milkmanId => json['milkman_id'] as String?;
  String? get price => json['price'] as String?;
  String? get productId => json['product_id'] as String?;
  String? get updatedAt => json['updated_at'] as String?;
}

class EventDto {
  final Map<String, dynamic> json;
  const EventDto(this.json);
  factory EventDto.fromJson(Map<String, dynamic> json) => EventDto(json);
  Map<String, dynamic> toJson() => json;
  String? get aggregateId => json['aggregate_id'] as String?;
  String? get aggregateType => json['aggregate_type'] as String?;
  int? get attemptCount => json['attempt_count'] as int?;
  String? get createdAt => json['created_at'] as String?;
  String? get eventType => json['event_type'] as String?;
  String? get id => json['id'] as String?;
  String? get lastError => json['last_error'] as String?;
  dynamic get payload => json['payload'];
  String? get publishedAt => json['published_at'] as String?;
  String? get status => json['status'] as String?;
}

class ExtraMilkRequestDto {
  final Map<String, dynamic> json;
  const ExtraMilkRequestDto(this.json);
  factory ExtraMilkRequestDto.fromJson(Map<String, dynamic> json) =>
      ExtraMilkRequestDto(json);
  Map<String, dynamic> toJson() => json;
  String? get baseQuantity => json['base_quantity'] as String?;
  String? get createdAt => json['created_at'] as String?;
  String? get customerId => json['customer_id'] as String?;
  String? get decidedAt => json['decided_at'] as String?;
  String? get decisionNote => json['decision_note'] as String?;
  String? get extraQuantity => json['extra_quantity'] as String?;
  String? get fullName => json['full_name'] as String?;
  String? get id => json['id'] as String?;
  String? get milkmanId => json['milkman_id'] as String?;
  String? get orderId => json['order_id'] as String?;
  String? get phone => json['phone'] as String?;
  String? get plannedQuantity => json['planned_quantity'] as String?;
  String? get productName => json['product_name'] as String?;
  String? get serviceDate => json['service_date'] as String?;
  String? get status => json['status'] as String?;
  String? get updatedAt => json['updated_at'] as String?;
}

class MFAChallengeDto {
  final Map<String, dynamic> json;
  const MFAChallengeDto(this.json);
  factory MFAChallengeDto.fromJson(Map<String, dynamic> json) =>
      MFAChallengeDto(json);
  Map<String, dynamic> toJson() => json;
  String? get mfaChallengeId => json['mfa_challenge_id'] as String?;
}

class MilkmanCustomerDto {
  final Map<String, dynamic> json;
  const MilkmanCustomerDto(this.json);
  factory MilkmanCustomerDto.fromJson(Map<String, dynamic> json) =>
      MilkmanCustomerDto(json);
  Map<String, dynamic> toJson() => json;
  String? get createdAt => json['created_at'] as String?;
  String? get deliveryAddress => json['delivery_address'] as String?;
  String? get fullName => json['full_name'] as String?;
  String? get id => json['id'] as String?;
  bool? get isActive => json['is_active'] as bool?;
  String? get phone => json['phone'] as String?;
  String? get phoneVerifiedAt => json['phone_verified_at'] as String?;
  String? get relationshipId => json['relationship_id'] as String?;
  String? get updatedAt => json['updated_at'] as String?;
}

class MilkmanRatingDto {
  final Map<String, dynamic> json;
  const MilkmanRatingDto(this.json);
  factory MilkmanRatingDto.fromJson(Map<String, dynamic> json) =>
      MilkmanRatingDto(json);
  Map<String, dynamic> toJson() => json;
  String? get createdAt => json['created_at'] as String?;
  String? get customerId => json['customer_id'] as String?;
  String? get id => json['id'] as String?;
  String? get milkmanId => json['milkman_id'] as String?;
  int? get rating => json['rating'] as int?;
  String? get updatedAt => json['updated_at'] as String?;
}

class OrderDto {
  final Map<String, dynamic> json;
  const OrderDto(this.json);
  factory OrderDto.fromJson(Map<String, dynamic> json) => OrderDto(json);
  Map<String, dynamic> toJson() => json;
  String? get approvedExtraQuantity =>
      json['approved_extra_quantity'] as String?;
  String? get baseQuantity => json['base_quantity'] as String?;
  String? get createdAt => json['created_at'] as String?;
  String? get customerId => json['customer_id'] as String?;
  String? get customerNote => json['customer_note'] as String?;
  String? get deliveryAddress => json['delivery_address'] as String?;
  String? get deliveryMarkedAt => json['delivery_marked_at'] as String?;
  String? get deliveryNote => json['delivery_note'] as String?;
  String? get id => json['id'] as String?;
  String? get markedBy => json['marked_by'] as String?;
  String? get notDeliveredReason => json['not_delivered_reason'] as String?;
  String? get priceSource => json['price_source'] as String?;
  String? get productId => json['product_id'] as String?;
  String? get quantity => json['quantity'] as String?;
  String? get sellerId => json['seller_id'] as String?;
  String? get serviceDate => json['service_date'] as String?;
  String? get status => json['status'] as String?;
  String? get totalAmount => json['total_amount'] as String?;
  String? get unitPrice => json['unit_price'] as String?;
  String? get updatedAt => json['updated_at'] as String?;
}

class ProductDto {
  final Map<String, dynamic> json;
  const ProductDto(this.json);
  factory ProductDto.fromJson(Map<String, dynamic> json) => ProductDto(json);
  Map<String, dynamic> toJson() => json;
  bool? get adminSuspended => json['admin_suspended'] as bool?;
  String? get createdAt => json['created_at'] as String?;
  String? get dailyCapacity => json['daily_capacity'] as String?;
  String? get id => json['id'] as String?;
  bool? get isAvailable => json['is_available'] as bool?;
  String? get name => json['name'] as String?;
  String? get price => json['price'] as String?;
  String? get sellerId => json['seller_id'] as String?;
  String? get unit => json['unit'] as String?;
  String? get updatedAt => json['updated_at'] as String?;
}

class PublicSellerDto {
  final Map<String, dynamic> json;
  const PublicSellerDto(this.json);
  factory PublicSellerDto.fromJson(Map<String, dynamic> json) =>
      PublicSellerDto(json);
  Map<String, dynamic> toJson() => json;
  String? get averageRating => json['average_rating'] as String?;
  String? get city => json['city'] as String?;
  String? get fullName => json['full_name'] as String?;
  String? get id => json['id'] as String?;
  String? get pincode => json['pincode'] as String?;
  int? get rank => json['rank'] as int?;
  int? get ratingCount => json['rating_count'] as int?;
}

class SummaryDto {
  final Map<String, dynamic> json;
  const SummaryDto(this.json);
  factory SummaryDto.fromJson(Map<String, dynamic> json) => SummaryDto(json);
  Map<String, dynamic> toJson() => json;
  int? get deliveredDays => json['delivered_days'] as int?;
  String? get deliveredLitres => json['delivered_litres'] as String?;
  dynamic get groups => json['groups'];
  String? get totalAmount => json['total_amount'] as String?;
}

class TokensDto {
  final Map<String, dynamic> json;
  const TokensDto(this.json);
  factory TokensDto.fromJson(Map<String, dynamic> json) => TokensDto(json);
  Map<String, dynamic> toJson() => json;
  String? get accessToken => json['access_token'] as String?;
  String? get refreshToken => json['refresh_token'] as String?;
  dynamic get user => json['user'];
}

class UserDto {
  final Map<String, dynamic> json;
  const UserDto(this.json);
  factory UserDto.fromJson(Map<String, dynamic> json) => UserDto(json);
  Map<String, dynamic> toJson() => json;
  String? get addressLine => json['address_line'] as String?;
  String? get approvalReason => json['approval_reason'] as String?;
  String? get approvalStatus => json['approval_status'] as String?;
  String? get approvedAt => json['approved_at'] as String?;
  String? get approvedBy => json['approved_by'] as String?;
  String? get city => json['city'] as String?;
  String? get createdAt => json['created_at'] as String?;
  String? get fullName => json['full_name'] as String?;
  String? get id => json['id'] as String?;
  bool? get isEnabled => json['is_enabled'] as bool?;
  String? get phone => json['phone'] as String?;
  String? get phoneVerifiedAt => json['phone_verified_at'] as String?;
  String? get pincode => json['pincode'] as String?;
  String? get role => json['role'] as String?;
  String? get updatedAt => json['updated_at'] as String?;
}
