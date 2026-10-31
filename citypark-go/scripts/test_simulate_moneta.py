import contextlib
import io
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import simulate_moneta as simulator


class SimulatorTests(unittest.TestCase):
    def test_published_pay_signature(self):
        fields = simulator.signed_fields(15, "100", "secret", "0", "555", "790.00")
        self.assertEqual(fields["MNT_SIGNATURE"], "998b6268fe3a6c7f4eeb8ccec65a9566")
        self.assertNotIn("MNT_COMMAND", fields)

    def test_published_check_signature(self):
        fields = simulator.signed_fields("FF790ABCD", "54600817", "QWERTY", "0",
                                         amount="120.25", command="CHECK")
        self.assertEqual(fields["MNT_SIGNATURE"], "ea2d49048bdf11857f1b50270aedbc8d")

    def response(self, code="100", amount="790.00"):
        signature = simulator.digest(code + "10016secret")
        return (f"<MNT_RESPONSE><MNT_ID>100</MNT_ID><MNT_TRANSACTION_ID>16</MNT_TRANSACTION_ID>"
                f"<MNT_RESULT_CODE>{code}</MNT_RESULT_CODE><MNT_AMOUNT>{amount}</MNT_AMOUNT>"
                f"<MNT_SIGNATURE>{signature}</MNT_SIGNATURE></MNT_RESPONSE>")

    def test_check_response_validation(self):
        self.assertEqual(simulator.checked_amount(self.response(), "100", 16, "secret"), "790.00")
        for body in [self.response("200"), self.response("500"), self.response(amount="NaN"),
                     self.response().replace("<MNT_ID>100", "<MNT_ID>101")]:
            with self.subTest(body=body), self.assertRaises(ValueError):
                simulator.checked_amount(body, "100", 16, "secret")

    def test_settings(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / ".env"
            path.write_text('# example\n' + 'export MONETA_ID="100"\nMONETA_SECRET=secret\n')
            with patch.dict(simulator.os.environ, {"MONETA_ID": "200"}, clear=True):
                settings = simulator.load_settings(path)
            self.assertEqual(settings["MONETA_ID"], "200")
            self.assertEqual(settings["MONETA_SECRET"], "secret")

    def test_cli_check_then_pay(self):
        with patch.dict(simulator.os.environ, {"MONETA_ID": "100", "MONETA_SECRET": "secret",
                                              "MONETA_TEST_MODE": "true"}, clear=True):
            with patch.object(simulator, "post", side_effect=[self.response(), "SUCCESS"]) as post:
                with contextlib.redirect_stdout(io.StringIO()):
                    self.assertEqual(simulator.main(["16", "--env-file", "/nonexistent"]), 0)
                self.assertTrue(post.call_args_list[0].args[0].endswith("/moneta/check"))
                fields = post.call_args_list[1].args[1]
                self.assertEqual(fields["MNT_TRANSACTION_ID"], "16")
                self.assertEqual(fields["MNT_AMOUNT"], "790.00")
                self.assertEqual(fields["MNT_TEST_MODE"], "1")
                self.assertEqual(fields["MNT_OPERATION_ID"], "simulated-16")

    def test_dry_run_does_not_pay(self):
        with patch.dict(simulator.os.environ, {"MONETA_ID": "100", "MONETA_SECRET": "secret"}, clear=True):
            with patch.object(simulator, "post") as post, contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(simulator.main(["16", "--amount", "790.00", "--dry-run",
                                                 "--env-file", "/nonexistent"]), 0)
                post.assert_not_called()


if __name__ == "__main__":
    unittest.main()
