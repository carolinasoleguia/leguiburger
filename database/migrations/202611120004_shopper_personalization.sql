ALTER TABLE legui_private.shoppers ADD COLUMN phone text NOT NULL DEFAULT '' CHECK(length(phone)<=30), ADD COLUMN default_address text NOT NULL DEFAULT '' CHECK(length(default_address)<=500);
ALTER TABLE legui_private.shopper_preferences ADD COLUMN personalization boolean NOT NULL DEFAULT false;
ALTER TABLE legui_private.shopper_preference_events ADD COLUMN personalization boolean NOT NULL DEFAULT false;
